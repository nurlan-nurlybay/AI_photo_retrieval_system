import asyncio
from functools import partial
from fastapi import FastAPI, HTTPException
from app.models import AddImageBatchRequest, AddTextBatchRequest, SearchRequest, DeleteItemsRequest
from app import core
from app.reranker import ranker
import json

app = FastAPI(title="Vector Service API")

@app.post("/v1/ingest/image")
async def ingest_image(req: AddImageBatchRequest):
    items = [{"id": item.id, "image_vector": item.image_vector} for item in req.items]
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, partial(core.add_image_batch, req.namespace, items))
    return {"ok": True, "status": f"inserted {len(items)} images"}

@app.post("/v1/ingest/text")
async def ingest_text(req: AddTextBatchRequest):
    items = [{"id": item.id, "text_vector": item.text_vector, "tags": item.tags} for item in req.items]
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, partial(core.add_text_batch, req.namespace, items))
    return {"ok": True, "status": f"inserted {len(items)} texts"}

@app.post("/v1/search/hybrid")
async def search_hybrid(req: SearchRequest):
    if not req.image_vector:
        raise HTTPException(status_code=400, detail="image_vector is required for baseline search")

    # 1. Always do baseline image search
    img_res = core.search_collection(f"{req.namespace}_img", req.image_vector, req.top_k * 2, is_text=False)
    img_hits = [{"id": hit.id, "score": hit.distance} for hit in img_res]
    
    final_results = img_hits
    used_qwen = False

    # 2. STRICT RULE: Only use Hybrid/Qwen if EVERY image in the namespace has a text vector
    if req.text_vector and core.check_sync_status(req.namespace):
        txt_res = core.search_collection(f"{req.namespace}_txt", req.text_vector, req.top_k * 2, is_text=True)
        
        if txt_res:
            txt_hits = [{"id": hit.id, "score": hit.distance, "tags": json.loads(hit.entity.get("tags"))} for hit in txt_res]
            
            # Safely handle the Optional[str] to make the type checker happy
            safe_query_text = req.query_text or ""
            
            final_results = ranker.rerank(safe_query_text, img_hits, txt_hits, top_k=req.top_k)
            used_qwen = True

    return {
        "results": final_results[:req.top_k],
        "used_qwen": used_qwen
    }

@app.get("/v1/status/{namespace}")
async def get_status(namespace: str):
    is_synced = core.check_sync_status(namespace)
    return {"namespace": namespace, "is_synced": is_synced}

@app.post("/v1/delete/items")
async def delete_items(req: DeleteItemsRequest):
    loop = asyncio.get_event_loop()
    await loop.run_in_executor(None, partial(core.delete_items, req.namespace, req.image_ids))
    return {"ok": True, "status": f"deleted {len(req.image_ids)} items from {req.namespace}"}

@app.post("/v1/admin/clear/{namespace}")
async def clear_namespace(namespace: str):
    core.clear_namespace_data(namespace)
    return {"ok": True, "status": f"Namespace {namespace} cleared."}

@app.post("/v1/admin/nuke")
async def nuke_system():
    deleted, errors = core.clear_all_namespaces()
    return {"ok": True, "deleted_collections": deleted, "errors": errors}
