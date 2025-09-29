# Vector Service (FAISS, in-memory)

## How to Run
```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8002
```

## Availabe Routes

### Upsert vector
```bash
curl -X POST http://localhost:8002/v1/vectors/add \
  -H "Content-Type: application/json" \
  -d '{ "namespace": "dataset:cifar10", "id": 123, "vector": [0.1, 0.2, ...] }'
```
```bash
curl -X POST http://localhost:8002/v1/vectors/add \
  -H "Content-Type: application/json" \
  -d '{"id":"media_2","vector":[0.11,0.19,0.29,0.41,0.52]}'
```


### Search Vector
```bash
curl -X POST http://localhost:8002/v1/vectors/search \
  -H "Content-Type: application/json" \
  -d '{ "namespace": "dataset:cifar10", "vector": [0.1, 0.2, ...], "k": 20 }
'
```
- Distance is cosine similarity (because vectors are L2-normalized and we use Inner Product).
- Switch to L2 by setting METRIC=l2 in env 

### Delete vector
```bash
curl -X DELETE http://localhost:8002/v1/vectors/media_1
```
Internally FAISS doesn’t “delete,” so the service either:
- Marks the ID in a tombstone set, or
- Rebuilds index periodically excluding deleted IDs.


### Notes 

- **Cosine by default.** We L2-normalize incoming vectors and use `IndexFlatIP`, which gives you cosine similarity in [0..1]. If ML already normalizes, fine, double normalization won’t hurt.
- **IDs are first-class.** We use `faiss.IndexIDMap2` and store your `photo_id` directly. No side maps.
- **Deletions work.** `POST /v1/vectors/remove` removes by ID using FAISS selectors.
- **Shards later.** The service exposes `SHARD_ID`. When you inevitably split indices by hash range or date, run multiple replicas behind Go orchestrator and fan-out queries.
