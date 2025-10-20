import threading
import faiss
import numpy as np
from typing import List, Dict, Any, Optional

# Global lock to keep FAISS thread-safe
LOCK = threading.RLock()

# Namespace (model) → FAISS index mapping
INDICES: Dict[str, faiss.Index] = {}

# Namespace (model) → FAISS index mapping
INDICES: Dict[str, faiss.Index] = {}

def _to_np(vec: List[float]) -> np.ndarray:
    """Convert list of floats to a 2D contiguous float32 numpy array."""
    arr = np.asarray(vec, dtype="float32")
    arr = np.atleast_2d(arr)  # ensures shape (1, dim)
    return np.ascontiguousarray(arr, dtype="float32")


def _normalize(arr: np.ndarray) -> np.ndarray:
    """L2-normalize rows for cosine similarity."""
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    norms[norms == 0] = 1.0
    return arr / norms


def _get_index(namespace: str, dim: int) -> faiss.Index:
    """Return existing index for namespace, or create a new one."""
    with LOCK:
        ix = INDICES.get(namespace)
        if ix is None:
            base = faiss.IndexFlatIP(dim)      # inner-product index
            ix = faiss.IndexIDMap2(base)       # supports external IDs
            INDICES[namespace] = ix
        else:
            if ix.d != dim:
                raise ValueError(f"vector_dim_mismatch: expected {ix.d}, got {dim}")
        return ix


def health() -> Dict[str, Any]:
    """Return a summary of all loaded FAISS indexes."""
    with LOCK:
        return {
            "namespaces": {ns: int(ix.ntotal) for ns, ix in INDICES.items()},
            "dims": {ns: ix.d for ns, ix in INDICES.items()},
        }


def add(namespace: str, media_id: int, vector: List[float], normalize: bool = True) -> Dict[str, Any]:
    """Add or update a vector in the specified FAISS namespace."""
    try:
        vec = _to_np(vector)
        if normalize:
            vec = _normalize(vec)

        ix = _get_index(namespace, vec.shape[1])
        faiss_id = np.asarray([np.int64(media_id)])

        with LOCK:
            # Idempotent upsert
            try:
                ix.remove_ids(faiss_id)
            except Exception:
                pass  # ignore missing IDs

            ix.add_with_ids(vec, faiss_id)

        return {
            "ok": True,
            "namespace": namespace,
            "id": media_id,
            "dim": ix.d,
        }

    except Exception as e:
        return {
            "ok": False,
            "namespace": namespace,
            "id": media_id,
            "error": str(e),
        }


def delete(namespace: str, media_id: int) -> Dict[str, Any]:
    """Remove a vector by ID from the specified namespace."""
    ix = INDICES.get(namespace)
    if ix is None:
        return {"ok": True, "namespace": namespace, "id": media_id, "deleted": False}

    with LOCK:
        n_removed = ix.remove_ids(np.asarray([np.int64(media_id)]))
        deleted = int(n_removed) > 0

    return {"ok": True, "namespace": namespace, "id": media_id, "deleted": deleted}


def search(namespace: str, vector: List[float], k: int, normalize: bool = True) -> Dict[str, Any]:
    """Search for top-k similar vectors in the specified namespace."""
    ix = INDICES.get(namespace)
    if ix is None or ix.ntotal == 0:
        return {"ok": True, "namespace": namespace, "k": 0, "results": [], "degraded": False}

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

    return {
        "ok": True,
        "namespace": namespace,
        "k": k,
        "results": results,
        "degraded": False,
    }
