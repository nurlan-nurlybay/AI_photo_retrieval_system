from pydantic import BaseModel, Field, field_validator
from typing import List, Optional

def validate_namespace(v: str) -> str:
    v = v.strip()
    if not v or v.lower() in ["default", "none", "null"]:
        raise ValueError("A specific user namespace is required. 'default' is not allowed.")
    return v

# ---------- Base ----------
class NamespaceRequest(BaseModel):
    namespace: str = Field(..., min_length=1, description="Unique user namespace (collection).")
    
    @field_validator('namespace')
    @classmethod
    def check_namespace(cls, v: str) -> str:
        return validate_namespace(v)

# ---------- Add ----------
class VectorAddRequest(NamespaceRequest):
    id: int = Field(..., description="Media ID")
    vector: List[float]
    normalize: bool = True

class VectorAddResponse(BaseModel):
    ok: bool
    id: int
    namespace: str
    dim: Optional[int] = None
    error: Optional[str] = None

# ---------- Delete ----------
class VectorDeleteRequest(NamespaceRequest):
    id: int

class VectorDeleteResponse(BaseModel):
    ok: bool
    id: int
    namespace: str
    deleted: bool
    error: Optional[str] = None

# ---------- Search ----------
class VectorSearchRequest(NamespaceRequest):
    vector: List[float]
    k: int = Field(..., gt=0, description="Number of results to return")
    normalize: bool = True

class SearchResult(BaseModel):
    id: int
    score: float

class VectorSearchResponse(BaseModel):
    ok: bool
    namespace: str
    k: int
    results: List[SearchResult] = Field(default_factory=list)
    error: Optional[str] = None

# ---------- Generic Responses ----------
class StandardResponse(BaseModel):
    ok: bool
    namespace: Optional[str] = None
    message: Optional[str] = None
    error: Optional[str] = None

class SystemClearResponse(BaseModel):
    ok: bool
    deleted_namespaces: int
    errors: List[str] = Field(default_factory=list)
