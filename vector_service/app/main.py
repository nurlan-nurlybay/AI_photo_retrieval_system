import json
from fastapi import FastAPI
from app.models import SearchRequest, AddImageRequest, AddTextRequest, NamespaceRequest
from app.reranker import ranker # Your existing w1, w2, w3 logic
from app import core

app = FastAPI()

@app.post("/v1/ingest/image")
async def ingest_image(req: AddImageRequest):
    core.add_image(req.namespace, req.id, req.image_vector)
    return {"ok": True, "status": "image_added"}

@app.post("/v1/ingest/text")
async def ingest_text(req: AddTextRequest):
    core.add_text(req.namespace, req.id, req.text_vector, req.tags)
    return {"ok": True, "status": "text_and_tags_added"}

@app.post("/v1/search/hybrid")
async def hybrid_search(req: SearchRequest):
    is_synced = core.check_sync_status(req.namespace)

    # 1. SEARCH IMAGES (Always happens)
    img_hits = core.search_collection(
        col_name=f"{req.namespace}_img", 
        vector=req.image_vector, 
        k=req.top_k * 2, 
        is_text=False
    )

    # --- FALLBACK PATH (Async workers still processing) ---
    if not is_synced:
        print(f"[{req.namespace}] Async workers lagging. Falling back to Pure SigLIP.")
        results = [{"id": hit.id, "score": hit.distance} for hit in img_hits]
        # Just sort by raw image distance (1.0 weight)
        results.sort(key=lambda x: x["score"], reverse=True)
        return results[:req.top_k]

    # --- HYBRID PATH (Dataset is fully processed) ---
    print(f"[{req.namespace}] Dataset fully synced. Using w1, w2, w3 Hybrid Search.")
    txt_hits = core.search_collection(
        col_name=f"{req.namespace}_txt", 
        vector=req.text_vector, 
        k=req.top_k * 2, 
        is_text=True
    )

    combined_data = {}
    for hit in img_hits:
        combined_data[hit.id] = {"img_sim": hit.distance, "txt_sim": 0.0, "tags": []}

    for hit in txt_hits:
        if hit.id in combined_data:
            combined_data[hit.id]["txt_sim"] = hit.distance
            # Deserialize tags from Milvus
            combined_data[hit.id]["tags"] = json.loads(hit.entity.get("tags"))
        else:
            combined_data[hit.id] = {
                "img_sim": 0.0, 
                "txt_sim": hit.distance, 
                "tags": json.loads(hit.entity.get("tags"))
            }

    final_rankings = []
    for media_id, data in combined_data.items():
        lex_score = ranker.calculate_lexical_score(req.query_text, data["tags"])
        final_score = ranker.get_hybrid_score(data["img_sim"], data["txt_sim"], lex_score)
        final_rankings.append({"id": media_id, "score": final_score})

    final_rankings.sort(key=lambda x: x["score"], reverse=True)
    return final_rankings[:req.top_k]


@app.delete("/v1/namespace/clear")
async def clear_namespace(req: NamespaceRequest):
    core.clear_namespace_data(req.namespace)
    return {"ok": True, "status": f"Namespace {req.namespace} cleared"}


@app.delete("/v1/system/nuke")
async def nuke_system():
    deleted, errors = core.clear_all_namespaces()
    return {"ok": True, "deleted_count": deleted, "errors": errors}
