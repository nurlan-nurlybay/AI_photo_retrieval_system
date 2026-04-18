import time
import asyncio
import httpx
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

# --- CONCURRENT NETWORK FETCHING ---
# Limit to 10 simultaneous S3 connections to avoid saturating the network
# interface on the Vast.ai instance (TCP/SSL handshake exhaustion).
_fetch_sem = asyncio.Semaphore(10)

async def fetch_url(client: httpx.AsyncClient, url: str) -> bytes:
    async with _fetch_sem:
        response = await client.get(url, timeout=30.0)
        response.raise_for_status()
        return response.content

@app.post("/v1/encode/image/url/fast/", response_model=VectorResponse)
async def url_fast_path(req: ImageURLRequest):
    if not req.urls:
        return {"vectors": []}

    async with httpx.AsyncClient() as client:
        tasks = [fetch_url(client, url) for url in req.urls]
        blobs = await asyncio.gather(*tasks)

    return {"vectors": encode_image_fast(blobs)}

@app.post("/v1/encode/image/url/slow/", response_model=SlowEncodeResponse)
async def url_slow_path(req: ImageURLRequest):
    if not req.urls:
        return {"results": []}

    async with httpx.AsyncClient() as client:
        tasks = [fetch_url(client, url) for url in req.urls]
        blobs = await asyncio.gather(*tasks)

    return {"results": encode_image_slow(blobs)}

@app.get("/healthz")
def healthz(): 
    return {"ok": True, "status": "healthy"}
