<<<<<<< HEAD
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


# ---------- Namespace Management ----------
class NamespaceListResponse(BaseModel):
    ok: bool
    namespaces: List[str] = Field(default_factory=list)
    count: int = 0
    error: Optional[str] = None


class NamespaceDeleteResponse(BaseModel):
    ok: bool
    namespace: str
    deleted: bool
    error: Optional[str] = None


class ClearAllResponse(BaseModel):
    ok: bool
    deleted_namespaces: int = 0
    total_namespaces: int = 0
    errors: List[str] = Field(default_factory=list)
    error: Optional[str] = None
=======
from pydantic import BaseModel
from typing import List, Optional

class NamespaceRequest(BaseModel):
    namespace: str

class ImageVectorItem(BaseModel):
    id: int
    image_vector: List[float]

class AddImageBatchRequest(NamespaceRequest):
    items: List[ImageVectorItem]

class TextVectorItem(BaseModel):
    id: int
    text_vector: List[float]
    tags: List[str]

class AddTextBatchRequest(NamespaceRequest):
    items: List[TextVectorItem]

class SearchRequest(NamespaceRequest):
    query_text: Optional[str] = None
    image_vector: Optional[List[float]] = None
    text_vector: Optional[List[float]] = None
    top_k: int = 10

class DeleteItemsRequest(NamespaceRequest):
    image_ids: List[int]
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
