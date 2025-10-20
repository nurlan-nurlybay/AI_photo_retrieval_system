## ML and Vector Services - Backend Integration Guide

### Endpoints and Base URLs
- Inside Docker network:
  - ml_service: `http://ml_service:8003`
  - vector_service: `http://vector_service:8002`
- From host (local testing):
  - ml_service: `http://localhost:8005`
  - vector_service: `http://localhost:8006`

Embedding dimension: 512. Keep normalization enabled for cosine similarity (the vector DB uses inner product/IP).

---

### ml_service API
Base path: `/v1`

1) POST `/encode/text/`
- Body JSON:
  - `req.texts` (required): list of strings, 1..1024 length
  - `options.model` (optional, default: `openai/clip-vit-base-patch32`)
  - `options.normalize` (optional, default: true)
  - `options.quantize` (optional, default: true)
- Response JSON:
  - `vectors`: list of 512-float vectors
- Example:
```
POST /v1/encode/text/
{
  "req": { "texts": ["a photo of a dog"] },
  "options": { "normalize": true, "quantize": true }
}
```

2) POST `/encode/image/`
- Multipart form; repeat `files=` to batch images
- Query params:
  - `model` (optional, default: `openai/clip-vit-base-patch32`)
  - `normalize` (optional, default: true)
- Response JSON:
  - `vectors`: list of 512-float vectors
- Example curl:
```
curl -X POST 'http://ml_service:8003/v1/encode/image/?model=openai/clip-vit-base-patch32&normalize=true' \
  -F 'files=@/path/to/img1.jpg' -F 'files=@/path/to/img2.jpg'
```

3) GET `/healthz`
- Response: `{ "ok": true }`

Notes:
- `quantize=true` speeds inference (2–4x) with negligible accuracy loss. Set `false` for full precision if needed.

---

### vector_service API
Base path: `/v1`

Namespaces map 1:1 to Milvus collections. Always pass a namespace string from the frontend/device. If unavailable, use a fallback like `"unresolved"`.

1) GET `/healthz`
- Response JSON:
```
{
  "ok": true,
  "connected": true,
  "host": "milvus",
  "port": "19530",
  "collections": {
    "<namespace>": { "num_entities": <int>, "dimension": 512 }
  }
}
```

2) POST `/vectors/add`
- Body JSON:
  - `namespace` (required): string (user/device identifier; e.g., "user123" or "unresolved")
  - `id` (required): integer media_id
  - `vector` (required): 512-float array
  - `normalize` (optional, default: true)
- Response JSON:
```
{
  "ok": true,
  "id": <int>,
  "namespace": "<string>",
  "replaced": false,
  "dim": 512,
  "error": null
}
```

3) POST `/vectors/search`
- Body JSON (backward compatible):
  - `namespace` (preferred) OR `model` (legacy) — one of them must be provided
  - `vector` (required): 512-float query vector
  - `k` (required): int > 0
  - `normalize` (optional, default: true)
- Response JSON:
```
{
  "ok": true,
  "namespace": "<string>",
  "k": <int>,
  "results": [ { "id": <int>, "score": <float> }, ... ],
  "degraded": false,
  "tookMs": null,
  "error": null
}
```

4) POST `/vectors/delete`
- Body JSON (backward compatible):
  - `namespace` (preferred) OR `model` (legacy)
  - `id` (required): int
- Response JSON:
```
{
  "ok": true,
  "namespace": "<string>",
  "id": <int>,
  "deleted": true
}
```

Semantics / Best Practices
- Always pass a `namespace` provided by the frontend (user/device id). If absent, use `"unresolved"`.
- Keep `normalize=true` unless you already normalized upstream; this maintains cosine similarity with IP in Milvus.
- Scores for text→image are typically ~0.2–0.35; absolute value matters less than ranking.
- For higher accuracy, try `openai/clip-vit-large-patch14` for both encoding and querying.

---

### Typical Backend Flow
1) Upload pipeline
   - Call `ml_service /v1/encode/image/` → get 512-d vector
   - POST to `vector_service /v1/vectors/add` with `{ namespace, id: media_id, vector, normalize: true }`

2) Search pipeline
   - If text query: `ml_service /v1/encode/text/` → take `vectors[0]`
   - POST to `vector_service /v1/vectors/search` with `{ namespace, vector, k, normalize: true }`
   - Return ranked ids to client

---

### Local Testing (host)
```
# Health
curl -s http://localhost:8005/healthz && echo ""   # ml
curl -s http://localhost:8006/v1/healthz | jq -c; echo ""

# Encode text
curl -s -X POST http://localhost:8005/v1/encode/text/ \
  -H "Content-Type: application/json" \
  -d '{"req":{"texts":["a photo of a dog"]}}' | tee text_vec.json

# Add vector (example)
curl -s -X POST http://localhost:8006/v1/vectors/add \
  -H "Content-Type: application/json" \
  -d '{"namespace":"user1","id":1,"vector":[...512 floats...],"normalize":true}' -w "\n"

# Search
curl -s -X POST http://localhost:8006/v1/vectors/search \
  -H "Content-Type: application/json" \
  -d "{\"namespace\":\"user1\",\"vector\":$(jq -c '.vectors[0]' text_vec.json),\"k\":3,\"normalize\":true}" \
  | jq -c; echo ""
```

---

### Operational Notes
- Services are available to the backend via Docker DNS names (`ml_service`, `vector_service`). Host ports (8005/8006) are for local testing only.
- Vector storage is persistent via Milvus (etcd + MinIO); vectors survive restarts.
- The vector index is configured for inner product (IP). For small datasets we use FLAT; for larger, consider IVF/HNSW and tune `nprobe` as needed.
