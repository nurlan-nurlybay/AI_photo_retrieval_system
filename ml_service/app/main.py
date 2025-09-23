from fastapi import FastAPI
from .models import TextRequest, VectorResponse
from .clip_service import encode_text

app = FastAPI()

@app.post("/v1/encode/text", response_model=VectorResponse)
def encode(req: TextRequest):
    vector = encode_text(req.text)
    return {"vector": vector}
