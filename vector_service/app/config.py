import os

<<<<<<< HEAD
# TODO: load from .env file
INDEX_DIM = int(os.getenv("INDEX_DIM", "512"))
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "8002"))
=======
MILVUS_HOST = os.getenv("MILVUS_HOST", "milvus-standalone")
MILVUS_PORT = os.getenv("MILVUS_PORT", "19530")
VECTOR_DIM = int(os.getenv("VECTOR_DIM", "1152"))
PORT = int(os.getenv("PORT", "8002"))
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
