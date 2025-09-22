from pydantic import BaseModel, conlist, Field
from typing import List, Literal, Optional

FloatVec = conlist(float, min_items=1)

class VectorItem(BaseModel):
    id: int = Field(..., ge=0)
    vector: FloatVec

class AddBatchRequest(BaseModel):
    items: List[VectorItem]

class AddOneRequest(BaseModel):
    id: int
    vector: FloatVec

class SearchRequest(BaseModel):
    vector: FloatVec
    k: int = 10

class RemoveRequest(BaseModel):
    ids: List[int]

class StatsResponse(BaseModel):
    count: int
    dim: int
    metric: Literal["cosine","l2"]
    shard_id: str
