from fastapi import FastAPI
from fastapi.responses import JSONResponse
from app.models import (
    VectorAddRequest, VectorAddResponse,
    VectorDeleteRequest, VectorDeleteResponse,
    VectorSearchRequest, VectorSearchResponse,
)
import app.index_milvus as idx
import traceback

app = FastAPI(title="faiss-service", version="1.0")

# ---------- Health ----------
@app.get("/v1/healthz")
def healthz():
    state = idx.health()
    return {"ok": True, **state}


# ---------- Add ----------
@app.post("/v1/vectors/add", response_model=VectorAddResponse)
def add_vector(req: VectorAddRequest):
    try:
        res = idx.add(req.namespace, req.id, req.vector, req.normalize)
        return VectorAddResponse(
            ok=True,
            id=req.id,
            namespace=req.namespace,
            replaced=False,
            dim=res.get("dim"),
        )

    except ValueError as e:
        return JSONResponse(
            status_code=400,
            content={"ok": False, "id": req.id, "namespace": req.namespace, "error": str(e)},
        )
    except Exception as e:
        traceback.print_exc()
        return JSONResponse(
            status_code=500,
            content={
                "ok": False,
                "id": req.id,
                "namespace": req.namespace,
                "error": str(e) or "index_write_failed",
            },
        )


# ---------- Delete ----------
@app.post("/v1/vectors/delete", response_model=VectorDeleteResponse)
def delete_vector(req: VectorDeleteRequest):
    try:
        res = idx.delete(req.model, req.id)
        return VectorDeleteResponse(
            ok=True,
            id=req.id,
            model=req.model,
            deleted=res.get("deleted", False),
        )

    except KeyError:
        return JSONResponse(
            status_code=404,
            content={"ok": False, "model": req.model, "id": req.id, "error": "not_found"},
        )
    except Exception as e:
        traceback.print_exc()
        return JSONResponse(
            status_code=500,
            content={
                "ok": False,
                "model": req.model,
                "id": req.id,
                "error": str(e) or "index_delete_failed",
            },
        )


# ---------- Search ----------
@app.post("/v1/vectors/search", response_model=VectorSearchResponse)
def search_vector(req: VectorSearchRequest):
    try:
        res = idx.search(req.model, req.vector, req.k, req.normalize)
        return VectorSearchResponse(
            ok=True,
            model=req.model,
            k=req.k,
            results=res.get("results", []),
            degraded=res.get("degraded", False),
            tookMs=res.get("tookMs"),
        )

    except ValueError as e:
        return JSONResponse(
            status_code=400,
            content={"ok": False, "model": req.model, "error": str(e), "k": req.k},
        )
    except Exception as e:
        traceback.print_exc()
        return JSONResponse(
            status_code=500,
            content={
                "ok": False,
                "model": req.model,
                "error": str(e) or "index_search_failed",
                "k": req.k,
            },
        )


# ---------- Entrypoint ----------
if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host="0.0.0.0", port=8002, reload=False)
