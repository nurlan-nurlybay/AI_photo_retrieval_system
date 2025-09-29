from pydantic import BaseModel, Field
from typing import List, Optional

# ---------- Add ----------
class VectorAddRequest(BaseModel):
    id: int
    vector: List[float]
    normalize: bool = True

class VectorAddResponse(BaseModel):
    ok: bool
    id: int
    replaced: bool = False
    dim: Optional[int] = None
    error: Optional[str] = None


# ---------- Delete ----------
class VectorDeleteResponse(BaseModel):
    ok: bool
    id: Optional[int] = None
    deleted: Optional[bool] = None
    error: Optional[str] = None


# ---------- Search ----------
class VectorSearchRequest(BaseModel):
    vector: List[float]
    k: int = Field(..., gt=0)
    normalize: bool = True

class SearchResult(BaseModel):
    id: int
    score: float

class VectorSearchResponse(BaseModel):
    ok: bool
    k: int
    results: List[SearchResult] = []
    degraded: bool = False
    tookMs: Optional[int] = None
    error: Optional[str] = None
