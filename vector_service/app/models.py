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
