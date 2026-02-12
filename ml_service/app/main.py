from contextlib import asynccontextmanager

from fastapi import FastAPI, UploadFile, File, Body, HTTPException
from .models import (
    TextRequest, VectorResponse,
    EncodeOptions, ModelName,
)
from .clip_service import (
    encode_text,
    encode_image,
    warmup,
)
import requests


@asynccontextmanager
async def lifespan(app: FastAPI):
    warmup()
    yield


app = FastAPI(lifespan=lifespan)

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


# ---------- Batch image (URLs) ----------
@app.post("/v1/encode/image/url/", response_model=VectorResponse)
def encode_image_urls(urls: list[str], options: EncodeOptions = Body(EncodeOptions())):
    # Download images from URLs
    image_bytes = []
    for url in urls:
        try:
            resp = requests.get(url, timeout=10)
            resp.raise_for_status()
            image_bytes.append(resp.content)
        except requests.RequestException as e:
            raise HTTPException(status_code=400, detail=f"Failed to download image from {url}: {str(e)}")

    return {"vectors": encode_image(image_bytes, options)}


# ---------- Healthcheck ----------
@app.get("/healthz")
def healthz(): return {"ok": True}
