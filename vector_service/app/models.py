from pydantic import BaseModel, Field
from typing import List, Optional

# ---------- Add ----------
class VectorAddRequest(BaseModel):
    model: str = "img-v1"    
    id: int                    # media_id
    vector: List[float]
    normalize: bool = True

class VectorAddResponse(BaseModel):
    ok: bool
    id: int
    model: Optional[str] = None
    replaced: bool = False
    dim: Optional[int] = None
    error: Optional[str] = None


# ---------- Delete ----------
class VectorDeleteRequest(BaseModel):
    """Remove a vector by ID from a specific namespace."""
    model: str = "img-v1"
    id: int


class VectorDeleteResponse(BaseModel):
    ok: bool
    id: Optional[int] = None
    model: Optional[str] = None
    deleted: Optional[bool] = None
    error: Optional[str] = None


# ---------- Search ----------
class VectorSearchRequest(BaseModel):
    model: str = "img-v1"
    vector: List[float]
    k: int = Field(..., gt=0)
    normalize: bool = True


class SearchResult(BaseModel):
    id: int
    score: float


class VectorSearchResponse(BaseModel):
    ok: bool
    model: Optional[str] = None
    k: int
    results: List[SearchResult] = Field(default_factory=list)
    degraded: bool = False
    tookMs: Optional[int] = None
    error: Optional[str] = None