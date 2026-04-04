import threading
import numpy as np
import structlog
from pymilvus import connections, Collection, FieldSchema, CollectionSchema, DataType, utility
from app.config import MILVUS_HOST, MILVUS_PORT, VECTOR_DIM

logger = structlog.get_logger(__name__)

# Global state
LOCK = threading.RLock()
_connected = False
_collections = {}

def connect():
    """Establish connection to Milvus."""
    global _connected
    if not _connected:
        try:
            connections.connect(alias="default", host=MILVUS_HOST, port=MILVUS_PORT)
            _connected = True
            logger.info("milvus_connected", host=MILVUS_HOST, port=MILVUS_PORT)
        except Exception as e:
            logger.error("milvus_connection_failed", error=str(e))
            raise

def get_collection(namespace: str) -> Collection:
    """Get or create a collection (namespace)."""
    connect()
    with LOCK:
        if namespace in _collections:
            return _collections[namespace]

        if utility.has_collection(namespace):
            collection = Collection(namespace)
        else:
            logger.info("creating_collection", namespace=namespace, dim=VECTOR_DIM)
            fields = [
                FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=False),
                FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=VECTOR_DIM)
            ]
            schema = CollectionSchema(fields, f"Namespace {namespace}")
            collection = Collection(namespace, schema)

            # Create IVF_FLAT index for Cosine Similarity (IP with normalized vectors)
            index_params = {
                "metric_type": "IP", 
                "index_type": "IVF_FLAT",
                "params": {"nlist": 1024}
            }
            collection.create_index("vector", index_params)

        collection.load()
        _collections[namespace] = collection
        return collection

def _normalize(vec: list[float]) -> np.ndarray:
    arr = np.asarray(vec, dtype="float32").reshape(1, -1)
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    norms = np.where(norms == 0, 1.0, norms)
    return np.clip(arr / norms, -1.0, 1.0)

# ---------- Core Operations ----------

def add_vector(namespace: str, media_id: int, vector: list[float], normalize: bool = True):
    vec = _normalize(vector) if normalize else np.asarray(vector, dtype="float32").reshape(1, -1)
    if vec.shape[1] != VECTOR_DIM:
        raise ValueError(f"Vector dimension mismatch. Expected {VECTOR_DIM}, got {vec.shape[1]}")
    
    with LOCK:
        col = get_collection(namespace)
        col.insert([[media_id], [vec[0].tolist()]])
        col.flush()

def delete_vector(namespace: str, media_id: int) -> bool:
    with LOCK:
        col = get_collection(namespace)
        mr = col.delete(f"id in [{media_id}]")
        return getattr(mr, "delete_count", 0) > 0

def search_vectors(namespace: str, vector: list[float], k: int, normalize: bool = True):
    vec = _normalize(vector) if normalize else np.asarray(vector, dtype="float32").reshape(1, -1)
    if vec.shape[1] != VECTOR_DIM:
        raise ValueError(f"Vector dimension mismatch. Expected {VECTOR_DIM}, got {vec.shape[1]}")

    with LOCK:
        col = get_collection(namespace)
        res = col.search(
            data=vec.tolist(),
            anns_field="vector",
            param={"metric_type": "IP", "params": {"nprobe": 16}},
            limit=k,
            output_fields=["id"]
        )
    
    hits = res[0] if res else []
    return [{"id": int(hit.id), "score": float(np.clip(hit.distance, 0.0, 1.0))} for hit in hits]

# ---------- Namespace Management ----------

def drop_namespace(namespace: str) -> bool:
    """Completely deletes the collection and all its data."""
    connect()
    with LOCK:
        if utility.has_collection(namespace):
            utility.drop_collection(namespace)
            _collections.pop(namespace, None)
            logger.info("namespace_deleted", namespace=namespace)
            return True
        return False

def clear_namespace_data(namespace: str):
    """Deletes all data but recreates an empty namespace."""
    drop_namespace(namespace)
    get_collection(namespace)
    logger.info("namespace_cleared", namespace=namespace)

def clear_all_namespaces() -> tuple[int, list[str]]:
    """Destroys every collection in Milvus."""
    connect()
    with LOCK:
        collections = utility.list_collections()
        deleted = 0
        errors = []
        for ns in collections:
            try:
                utility.drop_collection(ns)
                _collections.pop(ns, None)
                deleted += 1
            except Exception as e:
                errors.append(f"{ns}: {str(e)}")
        logger.warning("system_cleared", total_deleted=deleted)
        return deleted, errors
