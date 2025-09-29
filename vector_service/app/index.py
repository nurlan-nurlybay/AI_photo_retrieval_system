# vector_service/index.py

import threading
import faiss
import numpy as np
from typing import List, Dict, Any, Optional

DIM: Optional[int] = None
INDEX: Optional[faiss.Index] = None
LOCK = threading.RLock()

# namespace -> FAISS index mapping (simple, safer)
INDICES: Dict[str, faiss.Index] = {}

def _to_np(vec: List[float]) -> np.ndarray:
    arr = np.asarray(vec, dtype="float32")
    if arr.ndim == 1:
        arr = arr.reshape(1, -1)
    # faiss wants contiguous arrays
    return np.ascontiguousarray(arr, dtype="float32")

def _normalize(arr: np.ndarray) -> np.ndarray:
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    norms[norms == 0] = 1.0
    return arr / norms

def _get_index(ns: str, d: int) -> faiss.Index:
    with LOCK:
        ix = INDICES.get(ns)
        if ix is None:
            # FlatIP + IDMap: supports external int64 IDs directly
            base = faiss.IndexFlatIP(d)
            ix = faiss.IndexIDMap2(base)
            INDICES[ns] = ix
        else:
            if ix.d != d:
                raise ValueError(f"vector_dim_mismatch: expected {ix.d}, got {d}")
        return ix

def health() -> Dict[str, Any]:
    with LOCK:
        return {
            "namespaces": {ns: int(ix.ntotal) for ns, ix in INDICES.items()},
            "dims": {ns: ix.d for ns, ix in INDICES.items()},
        }

def add(ns: str, id: int, vector: List[float], normalize: bool = True) -> Dict[str, Any]:
    vec = _to_np(vector)
    if normalize:
        vec = _normalize(vec)
    ix = _get_index(ns, vec.shape[1])
    with LOCK:
        # remove existing id if present (idempotent upsert)
        faiss_id = np.asarray([np.int64(id)])
        try:
            ix.remove_ids(faiss_id)
        except Exception:
            pass  # ignore if not present
        ix.add_with_ids(vec, faiss_id)
        return {"ok": True, "namespace": ns, "id": id, "dim": ix.d}

def delete(ns: str, id: int) -> Dict[str, Any]:
    ix = INDICES.get(ns)
    if ix is None:
        return {"ok": True, "namespace": ns, "id": id, "deleted": False}
    with LOCK:
        n_removed = ix.remove_ids(np.asarray([np.int64(id)]))
        return {"ok": True, "namespace": ns, "id": id, "deleted": int(n_removed) > 0}

def search(ns: str, vector: List[float], k: int, normalize: bool = True) -> Dict[str, Any]:
    ix = INDICES.get(ns)
    if ix is None or ix.ntotal == 0:
        return {"ok": True, "namespace": ns, "k": 0, "results": [], "degraded": False}
    q = _to_np(vector)
    if normalize:
        q = _normalize(q)
    if q.shape[1] != ix.d:
        raise ValueError(f"vector_dim_mismatch: expected {ix.d}, got {q.shape[1]}")
    k = min(k, int(ix.ntotal))
    with LOCK:
        sims, ids = ix.search(q, k)
    results = [
        {"id": int(iid), "score": float(sim)}
        for sim, iid in zip(sims[0], ids[0])
        if iid != -1
    ]
    return {"ok": True, "namespace": ns, "k": k, "results": results, "degraded": False}
