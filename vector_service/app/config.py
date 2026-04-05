import os

MILVUS_HOST = os.getenv("MILVUS_HOST", "milvus-standalone")
MILVUS_PORT = os.getenv("MILVUS_PORT", "19530")
VECTOR_DIM = int(os.getenv("VECTOR_DIM", "1152"))
PORT = int(os.getenv("PORT", "8002"))