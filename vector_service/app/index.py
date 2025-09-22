import threading
import numpy as np
import faiss
from typing import List, Tuple
from .config import settings

def _np_float32(x) -> np.ndarray:
    arr = np.asarray(x, dtype=np.float32)
    if arr.ndim == 1:
        arr = arr.reshape(1, -1)
    return arr

def _l2_normalize(x: np.ndarray) -> np.ndarray:
    norms = np.linalg.norm(x, axis=1, keepdims=True) + 1e-12
    return x / norms

class VectorIndex:
    """
    Wraps a FAISS index with:
      - ID mapping (IndexIDMap2) so we store your photo_id directly
      - Cosine similarity using Inner Product on L2-normalized vectors
      - Thread-safety for FastAPI concurrency
    """
    def __init__(self, dim: int, metric: str = "cosine"):
        self.dim = dim
        self.metric = metric
        if metric == "cosine":
            base = faiss.IndexFlatIP(dim)
        elif metric == "l2":
            base = faiss.IndexFlatL2(dim)
        else:
            raise ValueError("Unsupported metric")
        self.index = faiss.IndexIDMap2(base)
        self._lock = threading.RLock()

    @property
    def count(self) -> int:
        with self._lock:
            return self.index.ntotal

    def add(self, ids: List[int], vectors: np.ndarray) -> None:
        if vectors.shape[1] != self.dim:
            raise ValueError(f"Vector dim {vectors.shape[1]} != index dim {self.dim}")
        if self.metric == "cosine":
            vectors = _l2_normalize(vectors)
        id_array = np.asarray(ids, dtype=np.int64)
        with self._lock:
            self.index.add_with_ids(vectors, id_array)

    def search(self, query: np.ndarray, k: int) -> Tuple[np.ndarray, np.ndarray]:
        if query.shape[1] != self.dim:
            raise ValueError(f"Query dim {query.shape[1]} != index dim {self.dim}")
        if self.metric == "cosine":
            query = _l2_normalize(query)
        with self._lock:
            D, I = self.index.search(query, k)
        return D, I

    def remove(self, ids: List[int]) -> int:
        sel = faiss.IDSelectorArray(np.asarray(ids, dtype=np.int64))
        with self._lock:
            removed = self.index.remove_ids(sel)
        # remove_ids returns the number of IDs removed
        return int(removed)

    def reset(self) -> None:
        with self._lock:
            self.index.reset()
