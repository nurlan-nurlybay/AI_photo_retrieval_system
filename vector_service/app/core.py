import faulthandler
import sys
import threading
import json
import numpy as np
import structlog

faulthandler.enable(file=sys.stderr)
from pymilvus import connections, Collection, FieldSchema, CollectionSchema, DataType, utility
from app.config import MILVUS_HOST, MILVUS_PORT, VECTOR_DIM

logger = structlog.get_logger(__name__)

LOCK = threading.RLock()
_connected = False
_collections = {}

def connect():
    global _connected
    if not _connected:
        connections.connect(alias="default", uri=f"http://{MILVUS_HOST}:{MILVUS_PORT}")
        _connected = True

def get_collection(col_name: str, is_text: bool = False) -> Collection:
    connect()
    with LOCK:
        if col_name in _collections:
            return _collections[col_name]

        if utility.has_collection(col_name):
            collection = Collection(col_name)
        else:
            fields = [
                FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=False),
                FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=VECTOR_DIM)
            ]
            if is_text:
                fields.append(FieldSchema(name="tags", dtype=DataType.VARCHAR, max_length=2048))

            schema = CollectionSchema(fields, f"Namespace {col_name}")
            collection = Collection(col_name, schema)

            index_params = {"metric_type": "IP", "index_type": "IVF_FLAT", "params": {"nlist": 1024}}
            collection.create_index("vector", index_params)  # type: ignore

        collection.load()
        _collections[col_name] = collection
        return collection

def _normalize(vec: list[float]) -> np.ndarray:
    arr = np.asarray(vec, dtype="float32").reshape(1, -1)
    norms = np.linalg.norm(arr, axis=1, keepdims=True)
    norms = np.where(norms == 0, 1.0, norms)
    return np.clip(arr / norms, -1.0, 1.0)

# --- BATCH INGESTION ---

def add_image_batch(namespace: str, items: list[dict]):
    if not items:
        return
    try:
        col = get_collection(f"{namespace}_img", is_text=False)
        
        data = [
            {"id": item["id"], "vector": _normalize(item["image_vector"])[0].tolist()}
            for item in items
        ]
        
        col.insert(data)
        col.flush()
        logger.info("image_batch_inserted", namespace=namespace, count=len(items))
    except Exception as e:
        logger.error("image_insert_failed", namespace=namespace, count=len(items), error=str(e))
        raise

def add_text_batch(namespace: str, items: list[dict]):
    if not items:
        return
    try:
        col = get_collection(f"{namespace}_txt", is_text=True)
        
        data = [
            {
                "id": item["id"], 
                "vector": _normalize(item["text_vector"])[0].tolist(),
                "tags": json.dumps(item["tags"])
            }
            for item in items
        ]
        
        col.insert(data)
        col.flush()
        logger.info("text_batch_inserted", namespace=namespace, count=len(items))
    except Exception as e:
        logger.error("text_insert_failed", namespace=namespace, count=len(items), error=str(e))
        raise

# --- SEARCH & SYNC CHECK ---

def check_sync_status(namespace: str) -> bool:
    img_col = get_collection(f"{namespace}_img", is_text=False)
    txt_col = get_collection(f"{namespace}_txt", is_text=True)
    img_col.flush()
    txt_col.flush()
    return img_col.num_entities == txt_col.num_entities

def search_collection(col_name: str, vector: list[float], k: int, is_text: bool) -> list:
    vec = _normalize(vector)
    col = get_collection(col_name, is_text=is_text)
    out_fields = ["id", "tags"] if is_text else ["id"]
    res = col.search(
        data=vec.tolist(),
        anns_field="vector",
        param={"metric_type": "IP", "params": {"nprobe": 16}},
        limit=k,
        output_fields=out_fields
    )
    return res[0] if res else []  # type: ignore

# --- DELETION ---

def clear_namespace_data(namespace: str):
    connect()
    img_col = f"{namespace}_img"
    txt_col = f"{namespace}_txt"
    with LOCK:
        if utility.has_collection(img_col):
            utility.drop_collection(img_col)  # type: ignore
            _collections.pop(img_col, None)
        if utility.has_collection(txt_col):
            utility.drop_collection(txt_col)  # type: ignore
            _collections.pop(txt_col, None)

def clear_all_namespaces() -> tuple[int, list]:
    connect()
    deleted = 0
    errors = []
    with LOCK:
        try:
            collections = utility.list_collections()
            for col in collections:
                utility.drop_collection(col)  # type: ignore
                _collections.pop(col, None)
                deleted += 1
        except Exception as e:
            errors.append(str(e))
    return deleted, errors
