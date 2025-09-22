from fastapi import FastAPI, HTTPException
from typing import List
import numpy as np

from .config import settings
from .models import (
    AddBatchRequest, AddOneRequest, SearchRequest, RemoveRequest, StatsResponse
)
from .index import VectorIndex, _np_float32

app = FastAPI(title="Vector Service", version="0.1.0")

# Single in-memory index per shard
INDEX = VectorIndex(dim=settings.VECTOR_DIM, metric=settings.METRIC)

@app.get("/v1/healthz")
def healthz():
    return {"ok": True, "shard_id": settings.SHARD_ID}

@app.get("/v1/stats", response_model=StatsResponse)
def stats():
    return StatsResponse(
        count=INDEX.count,
        dim=settings.VECTOR_DIM,
        metric=settings.METRIC,
        shard_id=settings.SHARD_ID,
    )

@app.post("/v1/vectors/add")
def add_batch(req: AddBatchRequest):
    if not req.items:
        raise HTTPException(status_code=400, detail="Empty items")
    ids: List[int] = [it.id for it in req.items]
    vecs = [_np_float32(it.vector) for it in req.items]
    X = np.vstack(vecs)
    if X.shape[1] != settings.VECTOR_DIM:
        raise HTTPException(status_code=400, detail=f"Bad dim {X.shape[1]}; expected {settings.VECTOR_DIM}")
    INDEX.add(ids, X)
    return {"added": len(ids)}

@app.post("/v1/vectors/add_one")
def add_one(req: AddOneRequest):
    X = _np_float32(req.vector)
    if X.shape[1] != settings.VECTOR_DIM:
        raise HTTPException(status_code=400, detail=f"Bad dim {X.shape[1]}; expected {settings.VECTOR_DIM}")
    INDEX.add([req.id], X)
    return {"added": 1}

@app.post("/v1/search")
def search(req: SearchRequest):
    k = min(max(1, req.k), settings.MAX_K)
    q = _np_float32(req.vector)
    if q.shape[1] != settings.VECTOR_DIM:
        raise HTTPException(status_code=400, detail=f"Bad dim {q.shape[1]}; expected {settings.VECTOR_DIM}")
    D, I = INDEX.search(q, k)
    # Return photo_id + distance for first (and only) query row
    results = []
    for d, i in zip(D[0].tolist(), I[0].tolist()):
        if i == -1:
            continue
        results.append({"id": int(i), "distance": float(d)})
    return {"results": results}

@app.post("/v1/vectors/remove")
def remove(req: RemoveRequest):
    removed = INDEX.remove(req.ids)
    return {"removed": removed}

@app.post("/v1/reset")
def reset():
    INDEX.reset()
    return {"ok": True}
