import os

MILVUS_HOST = os.getenv("MILVUS_HOST", "milvus")
MILVUS_PORT = os.getenv("MILVUS_PORT", "19530")

# CRITICAL: ML_Service uses SigLIP which outputs 1152. 
# 512 was for the old CLIP model.
VECTOR_DIM = int(os.getenv("VECTOR_DIM", "1152"))
PORT = int(os.getenv("PORT", "8002"))
