from pydantic import BaseSettings

class Settings(BaseSettings):
    # CLIP uses 512 by default; change if you swap encoders
    VECTOR_DIM: int = 512

    # Cosine similarity via Inner Product with L2-normalized vectors
    METRIC: str = "cosine"  # ["cosine", "l2"]

    # For future sharding/routing
    SHARD_ID: str = "shard-0"

    # Max results guardrail
    MAX_K: int = 200

settings = Settings()
