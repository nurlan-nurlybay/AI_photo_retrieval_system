# Vector Service (FAISS, in-memory)

## Run
```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8002
```

## Add vectors (batch)
```bash
curl -X POST http://localhost:8002/v1/vectors/add \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"id": 101, "vector": [0.1, 0.2, ... 512 floats ...]},
      {"id": 102, "vector": [0.05, 0.3, ...]}
    ]
  }'
```

## Search
```bash
curl -X POST http://localhost:8002/v1/search \
  -H "Content-Type: application/json" \
  -d '{"vector":[0.1,0.2,...],"k":5}'

```
Response: 
```bash
{"results":[{"id":101,"distance":0.83},{"id":102,"distance":0.79}]}
```
- Distance is cosine similarity (because vectors are L2-normalized and we use Inner Product).

- Switch to L2 by setting METRIC=l2 in env and restart.

### Notes you’ll pretend you read

- **Cosine by default.** We L2-normalize incoming vectors and use `IndexFlatIP`, which gives you cosine similarity in [0..1]. If your ML team already normalizes, fine, double normalization won’t explode.
- **IDs are first-class.** We use `faiss.IndexIDMap2` and store your `photo_id` directly. No silly side maps.
- **Deletions work.** `POST /v1/vectors/remove` removes by ID using FAISS selectors.
- **Shards later.** The service exposes `SHARD_ID`. When you inevitably split indices by hash range or date, run multiple replicas behind your Go orchestrator and fan-out queries. You’re welcome.
