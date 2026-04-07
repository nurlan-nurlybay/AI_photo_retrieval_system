<<<<<<< HEAD
from enum import Enum
from pydantic import BaseModel, conlist  # field_validator not used here

# ----- Batch text -----
class TextRequest(BaseModel):
    texts: conlist(str, min_length=1, max_length=1024)

class VectorResponse(BaseModel):
    # 512-d for ViT-B/32 and ViT-L/14
    vectors: list[conlist(float, min_length=512, max_length=512)]

# ----- Dynamic model/options -----
class ModelName(str, Enum):
    vit_b32 = "openai/clip-vit-base-patch32"
    vit_l14 = "openai/clip-vit-large-patch14"

class EncodeOptions(BaseModel):
    model: ModelName = ModelName.vit_b32
    normalize: bool = True
    quantize: bool = True
=======
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

# ----- S3 URL Request -----
class ImageURLRequest(BaseModel):
    urls: list[str]
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
