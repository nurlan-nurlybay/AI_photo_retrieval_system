<<<<<<< HEAD
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
=======
import time
import requests
import structlog
from contextlib import asynccontextmanager
from typing import List
from fastapi import FastAPI, Request, UploadFile, File
from prometheus_fastapi_instrumentator import Instrumentator

from .models import TextRequest, VectorResponse, SlowEncodeResponse, ImageURLRequest
from .ml_core import warmup, encode_image_slow, encode_image_fast, encode_text

structlog.configure(processors=[structlog.processors.JSONRenderer()])
logger = structlog.get_logger()

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Warmup SigLIP so search/fast-path is ready immediately
    warmup()
    yield

app = FastAPI(lifespan=lifespan)

instrumentator = Instrumentator().instrument(app).expose(app)

@app.middleware("http")
async def log_requests(request: Request, call_next):
    start_time = time.perf_counter()
    response = await call_next(request)
    duration = time.perf_counter() - start_time
    logger.info("http_request", method=request.method, path=request.url.path, duration=f"{duration:.4f}s")
    return response

@app.post("/v1/encode/text/", response_model=VectorResponse)
def encode_text_endpoint(req: TextRequest):
    return {"vectors": encode_text(req.texts)}

@app.post("/v1/encode/image/fast/", response_model=VectorResponse)
def fast_path(files: List[UploadFile] = File(...)):
    blobs = [f.file.read() for f in files]
    return {"vectors": encode_image_fast(blobs)}

@app.post("/v1/encode/image/slow/", response_model=SlowEncodeResponse)
def slow_path(files: List[UploadFile] = File(...)):
    blobs = [f.file.read() for f in files]
    return {"results": encode_image_slow(blobs)}

@app.post("/v1/encode/image/url/fast/", response_model=VectorResponse)
def url_fast_path(req: ImageURLRequest):
    blobs = []
    for url in req.urls:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        blobs.append(response.content)
    return {"vectors": encode_image_fast(blobs)}

@app.post("/v1/encode/image/url/slow/", response_model=SlowEncodeResponse)
def url_slow_path(req: ImageURLRequest):
    blobs = []
    for url in req.urls:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        blobs.append(response.content)
    return {"results": encode_image_slow(blobs)}

@app.get("/healthz")
def healthz(): 
    return {"ok": True, "status": "healthy"}
>>>>>>> aa3763fa7b72ca20a66743a7e808d3e539d2d5d1
