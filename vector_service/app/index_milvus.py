import threading
import numpy as np
from typing import List, Dict, Any

from .milvus_client import get_collection, health as milvus_health, VECTOR_DIM, list_collections, drop_collection

# Keep thread-safety similar to FAISS version
LOCK = threading.RLock()


def _to_np(vec: List[float]) -> np.ndarray:
    arr = np.asarray(vec, dtype="float32")
    arr = np.atleast_2d(arr)
    return np.ascontiguousarray(arr, dtype="float32")


def _normalize(arr: np.ndarray) -> np.ndarray:
    """Normalize vectors to unit length with numerical stability"""
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    # Avoid division by zero and ensure numerical stability
    norms = np.where(norms == 0, 1.0, norms)
    normalized = arr / norms
    # Clamp to prevent values slightly above 1.0 due to floating-point precision
    return np.clip(normalized, -1.0, 1.0)


def health() -> Dict[str, Any]:
    return milvus_health()


def add(namespace: str, media_id: int, vector: List[float], normalize: bool = True) -> Dict[str, Any]:
    try:
        vec = _to_np(vector)
        if normalize:
            vec = _normalize(vec)

        if vec.shape[1] != VECTOR_DIM:
            raise ValueError(f"vector_dim_mismatch: expected {VECTOR_DIM}, got {vec.shape[1]}")

        with LOCK:
            col = get_collection(namespace)
            # Insert expects column-wise data: [ids, [vector]]
            col.insert([[int(media_id)], [vec[0].tolist()]])
            col.flush()

        return {
            "ok": True,
            "namespace": namespace,
            "id": media_id,
            "dim": VECTOR_DIM,
        }

    except Exception as e:
        return {
            "ok": False,
            "namespace": namespace,
            "id": media_id,
            "error": str(e),
        }


def delete(namespace: str, media_id: int) -> Dict[str, Any]:
    with LOCK:
        col = get_collection(namespace)
        mr = col.delete(f"id in [{int(media_id)}]")
        deleted = False
        try:
            deleted = getattr(mr, "delete_count", 0) > 0  # pymilvus >=2.3
        except Exception:
            deleted = False

    return {"ok": True, "namespace": namespace, "id": media_id, "deleted": deleted}


def search(namespace: str, vector: List[float], k: int, normalize: bool = True) -> Dict[str, Any]:
    q = _to_np(vector)
    if normalize:
        q = _normalize(q)
    if q.shape[1] != VECTOR_DIM:
        raise ValueError(f"vector_dim_mismatch: expected {VECTOR_DIM}, got {q.shape[1]}")

    with LOCK:
        col = get_collection(namespace)
        col.load()
        res = col.search(
            data=q.tolist(),
            anns_field="vector",
            param={"metric_type": "IP", "params": {"nprobe": 16}},
            limit=int(k),
            output_fields=["id"],
        )

    hits = res[0] if res else []
    # Clamp scores to valid range [0, 1] to handle floating-point precision issues
    results = [{"id": int(hit.id), "score": float(np.clip(hit.distance, 0.0, 1.0))} for hit in hits]

    return {
        "ok": True,
        "namespace": namespace,
        "k": k,
        "results": results,
        "degraded": False,
    }


def list_namespaces() -> Dict[str, Any]:
    """List all available namespaces (collections)"""
    try:
        collections = list_collections()
        return {
            "ok": True,
            "namespaces": collections,
            "count": len(collections)
        }
    except Exception as e:
        return {
            "ok": False,
            "error": str(e),
            "namespaces": [],
            "count": 0
        }


def delete_namespace(namespace: str) -> Dict[str, Any]:
    """Delete a specific namespace (collection)"""
    try:
        with LOCK:
            dropped = drop_collection(namespace)
        return {
            "ok": True,
            "namespace": namespace,
            "deleted": dropped
        }
    except Exception as e:
        return {
            "ok": False,
            "namespace": namespace,
            "deleted": False,
            "error": str(e)
        }


def clear_all_data() -> Dict[str, Any]:
    """Clear all data (drop all collections)"""
    try:
        collections = list_collections()
        deleted_count = 0
        errors = []

        with LOCK:
            for namespace in collections:
                try:
                    if drop_collection(namespace):
                        deleted_count += 1
                except Exception as e:
                    errors.append(f"{namespace}: {str(e)}")

        return {
            "ok": True,
            "deleted_namespaces": deleted_count,
            "total_namespaces": len(collections),
            "errors": errors
        }
    except Exception as e:
        return {
            "ok": False,
            "error": str(e),
            "deleted_namespaces": 0,
            "total_namespaces": 0
        }
