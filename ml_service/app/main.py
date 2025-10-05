from fastapi import FastAPI, UploadFile, File, Body
from .models import (
    TextRequest, VectorResponse,
    EncodeOptions, ModelName,
)
from .clip_service import (
    encode_text,
    encode_image
)

app = FastAPI()

# ---------- Batch text (JSON body) ----------
@app.post("/v1/encode/text/", response_model=VectorResponse)
def encode_text_batch_endpoint(req: TextRequest, options: EncodeOptions = Body(EncodeOptions())) -> VectorResponse:
    return {"vectors": encode_text(req.texts, options)}


# ---------- Batch image (multipart + query params) ----------
@app.post("/v1/encode/image/", response_model=VectorResponse)
def encode_image_batch_endpoint(files: list[UploadFile] = File(...), model: ModelName = ModelName.vit_b32, normalize: bool = True) -> VectorResponse:
    options = EncodeOptions(model=model, normalize=normalize)
    blobs = [f.file.read() for f in files]  # images: list[bytes]
    return {"vectors": encode_image(blobs, options)}


# ---------- Healthcheck ----------
@app.get("/healthz")
def healthz(): return {"ok": True}
