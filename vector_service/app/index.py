import faiss
import numpy as np
from typing import List, Dict, Any

DIM: int | None = None
index: faiss.Index | None = None
id2int: Dict[str, int] = {}
int2id: Dict[int, str] = {}
_next_int_id: int = 1


def _ensure_index(d: int):
    global DIM, index
    if DIM is None:
        DIM = d
        index = faiss.IndexIDMap2(faiss.IndexFlatIP(DIM))


def _to_np(vec: List[float]) -> np.ndarray:
    arr = np.asarray(vec, dtype="float32")
    if arr.ndim == 1:
        arr = arr.reshape(1, -1)
    return arr


def _normalize(arr: np.ndarray) -> np.ndarray:
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    norms[norms == 0] = 1.0
    return arr / norms


def health() -> dict[str, Any]:
    return {
        "dim": DIM,
        "ntotal": int(index.ntotal) if index else 0,
    }


def add(id: str, vector: List[float], normalize: bool = True) -> dict[str, Any]:
    global _next_int_id
    vec = _to_np(vector)                # 1. turn Python list → NumPy float32 [1, D]
    _ensure_index(vec.shape[1])         # 2. if no FAISS index exists yet, build one with correct dim

    if vec.shape[1] != DIM:             # 3. sanity check — reject if dims don’t match index
        raise ValueError(...)
    if normalize:                       # 4. optional L2-normalize vector (needed for cosine sim)
        vec = _normalize(vec)

    replaced = False
    if id in id2int:                    # 5. if this ID already exists…
        old_int = id2int[id]
        index.remove_ids(np.asarray([old_int], dtype="int64"))  # …remove the old one from FAISS
        replaced = True
        del id2int[id]
        del int2id[old_int]

    int_id = _next_int_id               # 6. assign a fresh integer ID (FAISS wants int64 keys)
    _next_int_id += 1
    index.add_with_ids(vec, np.asarray([int_id], dtype="int64"))  # 7. stuff vector into FAISS

    id2int[id] = int_id                 # 8. update your lookup maps
    int2id[int_id] = id

    return {"ok": True, "id": id, "replaced": replaced, "dim": DIM}



def delete(id: str) -> dict[str, Any]:
    if id not in id2int:
        raise KeyError(id)
    int_id = id2int[id]
    index.remove_ids(np.asarray([int_id], dtype="int64"))
    del id2int[id]
    del int2id[int_id]
    return {"ok": True, "id": id, "deleted": True}


def search(vector: List[float], k: int, normalize: bool = True) -> dict[str, Any]:
    if index is None or index.ntotal == 0:
        return {"ok": True, "k": k, "results": [], "degraded": False, "tookMs": 0}

    q = _to_np(vector)
    if q.shape[1] != DIM:
        raise ValueError(f"vector_dim_mismatch: expected {DIM}, got {q.shape[1]}")
    if normalize:
        q = _normalize(q)

    k = min(k, int(index.ntotal))
    sims, ids = index.search(q, k)

    results = []
    for sim, iid in zip(sims[0], ids[0]):
        if iid == -1:
            continue
        sid = int2id.get(int(iid))
        if sid is None:
            continue
        results.append({"id": sid, "score": float(sim)})

    return {"ok": True, "k": k, "results": results, "degraded": False}
