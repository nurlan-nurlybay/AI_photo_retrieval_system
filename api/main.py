from fastapi import FastAPI, UploadFile, File, HTTPException, Response
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import requests, os, json, uuid

APP = FastAPI(title="Photo Retrieval Gateway")

APP.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],      # tighten later
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

ML = os.getenv("ML_URL", "http://ml_service:8003")
VS = os.getenv("VS_URL", "http://vector_service:8002")
FILER = os.getenv("FILER_URL", "http://seaweedfs:8888")         # internal to docker
PUBLIC_IMAGE_BASE = os.getenv("PUBLIC_IMAGE_BASE", "http://localhost:8888")  # what clients use
META_PATH = os.getenv("META_PATH", "/data/metadata_store.json")

# ---- tiny metadata store { id: {url,label} } ----
_meta = {}
def _load_meta():
    global _meta
    try:
        with open(META_PATH, "r") as f:
            _meta = json.load(f)
    except FileNotFoundError:
        _meta = {}
def _save_meta():
    os.makedirs(os.path.dirname(META_PATH), exist_ok=True)
    with open(META_PATH, "w") as f:
        json.dump(_meta, f)
_load_meta()

def _guess_ext(content_type: str | None) -> str:
    if not content_type: return ".jpg"
    ct = content_type.lower()
    if "png" in ct: return ".png"
    if "webp" in ct: return ".webp"
    return ".jpg"

class SearchTextReq(BaseModel):
    query: str
    top_k: int = 12

@APP.post("/v1/photos")
async def add_photo(file: UploadFile = File(...), label: str | None = None):
    # 0) read once
    content = await file.read()
    if not content:
        raise HTTPException(400, "empty file")

    # 1) embed
    try:
        vec = requests.post(f"{ML}/v1/encode/image",
                            files={"file": (file.filename, content, file.content_type)},
                            timeout=30).json()["vector"]
    except Exception as e:
        raise HTTPException(502, f"ml_service failed: {e}")

    # 2) add to vector index
    item_id = str(uuid.uuid4())
    try:
        requests.post(f"{VS}/v1/vectors/add",
                      json={"id": item_id, "vector": vec},
                      timeout=15).raise_for_status()
    except Exception as e:
        raise HTTPException(502, f"vector_service failed: {e}")

    # 3) upload to SeaweedFS filer
    ext = _guess_ext(file.content_type)
    filer_path = f"/photos/{item_id}{ext}"
    try:
        # Filer accepts multipart form: key must be "file"
        r = requests.post(f"{FILER}{filer_path}",
                          files={"file": (file.filename or f"{item_id}{ext}", content, file.content_type or "image/jpeg")},
                          timeout=30)
        r.raise_for_status()
    except Exception as e:
        raise HTTPException(502, f"filer upload failed: {e}")

    # 4) save metadata with public URL
    url = f"{PUBLIC_IMAGE_BASE}{filer_path}"
    _meta[item_id] = {"url": url, "label": label or (file.filename or item_id)}
    _save_meta()

    return {"ok": True, "id": item_id, "url": url, "label": _meta[item_id]["label"]}

@APP.post("/v1/search/image")
async def search_image(file: UploadFile = File(...), k: int = 12):
    content = await file.read()
    if not content:
        raise HTTPException(400, "empty file")
    try:
        vec = requests.post(f"{ML}/v1/encode/image",
                            files={"file": (file.filename, content, file.content_type)},
                            timeout=30).json()["vector"]
        res = requests.post(f"{VS}/v1/vectors/search",
                            json={"vector": vec, "k": int(k)}, timeout=15).json()
    except Exception as e:
        raise HTTPException(502, f"upstream failed: {e}")

    hits = res.get("results", [])
    out = []
    for h in hits:
        m = _meta.get(h["id"], {})
        out.append({
            "id": h["id"],
            "score": h.get("score"),
            "url": m.get("url"),
            "label": m.get("label"),
        })
    return {"ok": True, "results": out}

@APP.post("/v1/search/text")
def search_text(req: SearchTextReq):
    try:
        vec = requests.post(f"{ML}/v1/encode/text", json={"text": req.query}, timeout=20).json()["vector"]
        res = requests.post(f"{VS}/v1/vectors/search", json={"vector": vec, "k": req.top_k}, timeout=15).json()
    except Exception as e:
        raise HTTPException(502, f"upstream failed: {e}")
    out = []
    for h in res.get("results", []):
        m = _meta.get(h["id"], {})
        out.append({"id": h["id"], "score": h.get("score"), "url": m.get("url"), "label": m.get("label")})
    return {"ok": True, "results": out}

@APP.get("/healthz")
def health():
    return {"ok": True}
