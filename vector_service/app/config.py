import os

# TODO: load from .env file
INDEX_DIM = int(os.getenv("INDEX_DIM", "512"))
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "8002"))
