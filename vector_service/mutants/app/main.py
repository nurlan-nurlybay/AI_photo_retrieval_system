import time
import structlog
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from prometheus_fastapi_instrumentator import Instrumentator

from app.models import (
    VectorAddRequest, VectorAddResponse,
    VectorDeleteRequest, VectorDeleteResponse,
    VectorSearchRequest, VectorSearchResponse,
    StandardResponse, SystemClearResponse
)
from app.core import (
    add_vector, delete_vector, search_vectors,
    drop_namespace, clear_namespace_data, clear_all_namespaces
)

# Setup Logging
structlog.configure(processors=[structlog.processors.JSONRenderer()])
logger = structlog.get_logger()

app = FastAPI(title="Vector Service", version="2.0")

# Setup Metrics
instrumentator = Instrumentator().instrument(app).expose(app)
from typing import Annotated
from typing import Callable
from typing import ClassVar

MutantDict = Annotated[dict[str, Callable], "Mutant"] # type: ignore


def _mutmut_trampoline(orig, mutants, call_args, call_kwargs, self_arg = None): # type: ignore
    """Forward call to original or mutated function, depending on the environment"""
    import os # type: ignore
    mutant_under_test = os.environ['MUTANT_UNDER_TEST'] # type: ignore
    if mutant_under_test == 'fail': # type: ignore
        from mutmut.__main__ import MutmutProgrammaticFailException # type: ignore
        raise MutmutProgrammaticFailException('Failed programmatically')       # type: ignore
    elif mutant_under_test == 'stats': # type: ignore
        from mutmut.__main__ import record_trampoline_hit # type: ignore
        record_trampoline_hit(orig.__module__ + '.' + orig.__name__) # type: ignore
        # (for class methods, orig is bound and thus does not need the explicit self argument)
        result = orig(*call_args, **call_kwargs) # type: ignore
        return result # type: ignore
    prefix = orig.__module__ + '.' + orig.__name__ + '__mutmut_' # type: ignore
    if not mutant_under_test.startswith(prefix): # type: ignore
        result = orig(*call_args, **call_kwargs) # type: ignore
        return result # type: ignore
    mutant_name = mutant_under_test.rpartition('.')[-1] # type: ignore
    if self_arg is not None: # type: ignore
        # call to a class method where self is not bound
        result = mutants[mutant_name](self_arg, *call_args, **call_kwargs) # type: ignore
    else:
        result = mutants[mutant_name](*call_args, **call_kwargs) # type: ignore
    return result # type: ignore

@app.middleware("http")
async def log_requests(request: Request, call_next):
    start = time.perf_counter()
    response = await call_next(request)
    duration = time.perf_counter() - start
    logger.info("http_request", method=request.method, path=request.url.path, status=response.status_code, duration=f"{duration:.4f}s")
    return response

@app.get("/healthz")
def healthz():
    return {"ok": True, "status": "healthy"}

# ---------- Vector Operations ----------

@app.post("/v1/vectors/add", response_model=VectorAddResponse)
def api_add_vector(req: VectorAddRequest):
    try:
        add_vector(req.namespace, req.id, req.vector, req.normalize)
        return {"ok": True, "id": req.id, "namespace": req.namespace, "dim": len(req.vector)}
    except Exception as e:
        logger.error("vector_add_failed", namespace=req.namespace, id=req.id, error=str(e))
        return JSONResponse(status_code=500, content={"ok": False, "id": req.id, "namespace": req.namespace, "error": str(e)})

@app.post("/v1/vectors/delete", response_model=VectorDeleteResponse)
def api_delete_vector(req: VectorDeleteRequest):
    try:
        deleted = delete_vector(req.namespace, req.id)
        return {"ok": True, "id": req.id, "namespace": req.namespace, "deleted": deleted}
    except Exception as e:
        logger.error("vector_delete_failed", namespace=req.namespace, id=req.id, error=str(e))
        return JSONResponse(status_code=500, content={"ok": False, "id": req.id, "namespace": req.namespace, "deleted": False, "error": str(e)})

@app.post("/v1/vectors/search", response_model=VectorSearchResponse)
def api_search_vectors(req: VectorSearchRequest):
    try:
        results = search_vectors(req.namespace, req.vector, req.k, req.normalize)
        return {"ok": True, "namespace": req.namespace, "k": req.k, "results": results}
    except Exception as e:
        logger.error("vector_search_failed", namespace=req.namespace, error=str(e))
        return JSONResponse(status_code=500, content={"ok": False, "namespace": req.namespace, "k": req.k, "error": str(e)})

# ---------- System & Namespace Operations ----------

@app.post("/v1/namespaces/{namespace}/clear", response_model=StandardResponse)
def api_clear_dataset(namespace: str):
    """Wipes all vectors in a namespace but keeps the namespace ready for new data."""
    try:
        clear_namespace_data(namespace)
        return {"ok": True, "namespace": namespace, "message": "Dataset cleared."}
    except Exception as e:
        logger.error("clear_dataset_failed", namespace=namespace, error=str(e))
        return JSONResponse(status_code=500, content={"ok": False, "namespace": namespace, "error": str(e)})

@app.delete("/v1/namespaces/{namespace}", response_model=StandardResponse)
def api_delete_namespace(namespace: str):
    """Completely destroys the namespace/collection."""
    try:
        deleted = drop_namespace(namespace)
        msg = "Namespace deleted." if deleted else "Namespace not found."
        return {"ok": True, "namespace": namespace, "message": msg}
    except Exception as e:
        logger.error("delete_namespace_failed", namespace=namespace, error=str(e))
        return JSONResponse(status_code=500, content={"ok": False, "namespace": namespace, "error": str(e)})

@app.delete("/v1/system/clear-all", response_model=SystemClearResponse)
def api_clear_system():
    """DANGER: Destroys every collection in the Milvus database."""
    try:
        deleted, errors = clear_all_namespaces()
        return {"ok": len(errors) == 0, "deleted_namespaces": deleted, "errors": errors}
    except Exception as e:
        logger.error("system_clear_failed", error=str(e))
        return JSONResponse(status_code=500, content={"ok": False, "deleted_namespaces": 0, "errors": [str(e)]})

if __name__ == "__main__":
    import uvicorn
    from app.config import PORT
    uvicorn.run("app.main:app", host="0.0.0.0", port=PORT, reload=False)
