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
