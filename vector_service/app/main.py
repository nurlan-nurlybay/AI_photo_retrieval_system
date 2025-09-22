from fastapi import FastAPI
from fastapi.responses import JSONResponse
from app.models import (
    VectorAddRequest, VectorAddResponse,
    VectorDeleteResponse,
    VectorSearchRequest, VectorSearchResponse,
)
import app.index as idx

app = FastAPI(title="faiss-service", version="1.0")

@app.get("/v1/healthz")
def healthz():
    state = idx.health()
    return {"ok": True, **state}

@app.post("/v1/vectors/add", response_model=VectorAddResponse)
def add_vector(req: VectorAddRequest):
    try:
        res = idx.add(req.id, req.vector, req.normalize)
        return VectorAddResponse(**res)
    except ValueError as e:
        return JSONResponse(status_code=400, content={"ok": False, "error": str(e)})
    except Exception:
        return JSONResponse(status_code=500, content={"ok": False, "error": "index_write_failed"})

@app.delete("/v1/vectors/{id}", response_model=VectorDeleteResponse)
def delete_vector(id: str):
    try:
        res = idx.delete(id)
        return VectorDeleteResponse(**res)
    except KeyError:
        return JSONResponse(status_code=404, content={"ok": False, "error": "not_found", "id": id})
    except Exception:
        return JSONResponse(status_code=500, content={"ok": False, "error": "index_delete_failed"})

@app.post("/v1/vectors/search", response_model=VectorSearchResponse)
def search_vector(req: VectorSearchRequest):
    try:
        res = idx.search(req.vector, req.k, req.normalize)
        return VectorSearchResponse(**res)
    except ValueError as e:
        return JSONResponse(status_code=400, content={"ok": False, "error": str(e), "k": req.k})
    except Exception:
        return JSONResponse(status_code=500, content={"ok": False, "error": "index_search_failed", "k": req.k})

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("app.main:app", host="0.0.0.0", port=8002, reload=False)
