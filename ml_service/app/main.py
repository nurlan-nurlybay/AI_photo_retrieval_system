from fastapi import FastAPI, UploadFile, File
from .models import TextRequest, VectorResponse
from .clip_service import encode_text, encode_image

app = FastAPI()

@app.post("/v1/encode/text", response_model=VectorResponse)
def encode_text_endpoint(req: TextRequest):
    return {"vector": encode_text(req.text)}

@app.post("/v1/encode/image", response_model=VectorResponse)
def encode_image_endpoint(file: UploadFile = File(...)):
    return {"vector": encode_image(file.file.read())}