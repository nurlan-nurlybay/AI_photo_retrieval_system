from pydantic import BaseModel, Field, field_validator
from typing import List, Optional

def validate_namespace(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required.")
    return v

class NamespaceRequest(BaseModel):
    namespace: str = Field(..., min_length=1)
    
    @field_validator('namespace')
    @classmethod
    def check_namespace(cls, v: str) -> str:
        return validate_namespace(v)

# Fast Worker Payload
class AddImageRequest(NamespaceRequest):
    id: int
    image_vector: List[float]

# Slow Worker Payload
class AddTextRequest(NamespaceRequest):
    id: int
    text_vector: List[float]
    tags: List[str]

class SearchRequest(NamespaceRequest):
    query_text: str
    image_vector: List[float]
    text_vector: List[float]
    top_k: int = 10
