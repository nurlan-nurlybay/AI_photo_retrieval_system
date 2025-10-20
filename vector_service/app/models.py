from pydantic import BaseModel, Field
from typing import List, Optional

# ---------- Add ----------
class VectorAddRequest(BaseModel):
    namespace: str             # unique username/namespace
    id: int                    # media_id
    vector: List[float]
    normalize: bool = True

class VectorAddResponse(BaseModel):
    ok: bool
    id: int
    namespace: str
    replaced: bool = False
    dim: Optional[int] = None
    error: Optional[str] = None


# ---------- Delete ----------
class VectorDeleteRequest(BaseModel):
    """Remove a vector by ID from a specific namespace.
    Accepts either 'namespace' (preferred) or legacy 'model' for backward compatibility.
    """
    namespace: Optional[str] = None
    model: Optional[str] = None
    id: int


class VectorDeleteResponse(BaseModel):
    ok: bool
    id: Optional[int] = None
    namespace: str | None = None
    deleted: Optional[bool] = None
    error: Optional[str] = None


# ---------- Search ----------
class VectorSearchRequest(BaseModel):
    """Search within a namespace. Accepts 'namespace' or legacy 'model'."""
    namespace: Optional[str] = None
    model: Optional[str] = None
    vector: List[float]
    k: int = Field(..., gt=0)
    normalize: bool = True


class SearchResult(BaseModel):
    id: int
    score: float


class VectorSearchResponse(BaseModel):
    ok: bool
    namespace: str
    k: int
    results: List[SearchResult] = Field(default_factory=list)
    degraded: bool = False
    tookMs: Optional[int] = None
    error: Optional[str] = None
