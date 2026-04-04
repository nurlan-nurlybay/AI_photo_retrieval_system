import time
import structlog
from contextlib import asynccontextmanager
from typing import List
from fastapi import FastAPI, Request, UploadFile, File
from prometheus_fastapi_instrumentator import Instrumentator

from .models import TextRequest, VectorResponse, SlowEncodeResponse
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

@app.get("/healthz")
def healthz(): 
    return {"ok": True, "status": "healthy"}
