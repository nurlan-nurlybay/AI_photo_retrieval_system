from enum import Enum
from typing import Annotated
from pydantic import BaseModel, conlist, Field

# ----- Batch text -----
class TextRequest(BaseModel):
    texts: Annotated[list[str], Field(min_length=1, max_length=1024)]

class VectorResponse(BaseModel):
    # 512-d for ViT-B/32 and ViT-L/14
    vectors: Annotated[list[list[float]], Field(min_length=512, max_length=512)]

# ----- Dynamic model/options -----
class ModelName(str, Enum):
    vit_b32 = "openai/clip-vit-base-patch32"
    vit_l14 = "openai/clip-vit-large-patch14"

class EncodeOptions(BaseModel):
    model: ModelName = ModelName.vit_b32
    normalize: bool = True
    quantize: bool = True
