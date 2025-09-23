from pydantic import BaseModel

class TextRequest(BaseModel):
    text: str

class VectorResponse(BaseModel):
    vector: list[float]
