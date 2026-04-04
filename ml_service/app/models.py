from pydantic import BaseModel, Field
from typing import Annotated

# ----- Text Query -----
class TextRequest(BaseModel):
    texts: Annotated[list[str], Field(min_length=1, max_length=1024)]

# ----- Fast Response (SigLIP) -----
class VectorResponse(BaseModel):
    # SigLIP-so400m outputs 1152-dimensional vectors
    vectors: list[list[float]]

# ----- Slow Response (Qwen + SigLIP Text Vector) -----
class SlowEncodeResult(BaseModel):
    description: str
    tags: list[str]
    text_vector: list[float]

class SlowEncodeResponse(BaseModel):
    results: list[SlowEncodeResult]

