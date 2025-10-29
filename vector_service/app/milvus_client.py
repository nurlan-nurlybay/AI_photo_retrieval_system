import os
from pymilvus import (
    connections, Collection, FieldSchema, CollectionSchema, DataType,
    utility
)

# Milvus connection settings
MILVUS_HOST = os.getenv("MILVUS_HOST", "milvus")
MILVUS_PORT = os.getenv("MILVUS_PORT", "19530")
VECTOR_DIM = 512  # CLIP vector dimension

# Global connection state
_connected = False
_collections: dict[str, Collection] = {}


def connect() -> None:
    """Establish connection to Milvus server."""
    global _connected
    if not _connected:
        connections.connect(
            alias="default",
            host=MILVUS_HOST,
            port=MILVUS_PORT
        )
        _connected = True


def get_collection(namespace: str) -> Collection:
    """
    Get or create a collection (namespace) for vectors.
    Each namespace becomes a separate Milvus collection.
    """
    connect()

    if namespace in _collections:
        return _collections[namespace]

    # Check if collection exists
    if utility.has_collection(namespace):
        collection = Collection(namespace)
    else:
        # Create new collection
        fields = [
            FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=False),
            FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=VECTOR_DIM)
        ]
        schema = CollectionSchema(fields, f"Collection for namespace {namespace}")
        collection = Collection(namespace, schema)

        # Create index for fast search
        index_params = {
            "metric_type": "IP",  # Inner Product (cosine similarity with normalized vectors)
            "index_type": "IVF_FLAT",
            "params": {"nlist": 1024}
        }
        collection.create_index("vector", index_params)

    _collections[namespace] = collection
    return collection


def health() -> dict:
    """Check Milvus connection and return collection stats."""
    try:
        connect()

        # Get basic connection info
        info = {
            "connected": _connected,
            "host": MILVUS_HOST,
            "port": MILVUS_PORT,
            "collections": {}
        }

        # Get collection stats
        for namespace, collection in _collections.items():
            collection.load()  # Ensure collection is loaded
            info["collections"][namespace] = {
                "num_entities": collection.num_entities,
                "dimension": VECTOR_DIM
            }

        return info

    except Exception as e:
        return {
            "connected": False,
            "error": str(e),
            "collections": {}
        }


def list_collections() -> list[str]:
    """List all available collections (namespaces)"""
    connect()
    try:
        return utility.list_collections()
    except Exception:
        return []


def drop_collection(namespace: str) -> bool:
    """Drop a specific collection (namespace)"""
    connect()
    try:
        if utility.has_collection(namespace):
            utility.drop_collection(namespace)
            # Remove from our cache
            if namespace in _collections:
                del _collections[namespace]
            return True
        return False
    except Exception:
        return False
