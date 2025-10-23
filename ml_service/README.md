# ML Service (CLIP)

## How to Run
```bash
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8003
```

## Availabe Routes

### Encode Text into Vector

**Request**
```bash
curl -X POST http://localhost:8005/v1/encode/text/ \
  -H "Content-Type: application/json" \
  -d '{"text":"a red car"}'
```
**Response**
```json
{
  "vector": [0.0123, -0.0456, 0.0789, ...]
}
```



### Notes

Openai/clip-vit-base-patch32 model (pretrained, no extra training needed). Outputs 512-dimensional vector that are L2-normalized so FAISS inner product ≈ cosine similarity.
