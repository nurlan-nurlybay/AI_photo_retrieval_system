import json
import requests

# Forcing IPv4 to bypass Docker's localhost/IPv6 proxy blackhole
VECTOR_URL = "http://127.0.0.1:8006/v1"
JSON_FILE = "../test_running_containers/test_ml_service/ml_service_harvest.json"
NAMESPACE = "nurlan_gallery_batch"

def run_payload():
    with open(JSON_FILE, 'r') as f:
        data = json.load(f)

    print(f"Loaded JSON. Preparing batch of {len(data['gallery'])} items...")

    image_batch = []
    text_batch = []
    
    # Pack the lists
    for item in data["gallery"]:
        image_batch.append({"id": item["id"], "image_vector": item["image_vector"]})
        text_batch.append({"id": item["id"], "text_vector": item["text_vector"], "tags": item["tags"]})
        
    # 1. Ingest Image Batch
    print("Sending Image Batch...")
    res_img = requests.post(f"{VECTOR_URL}/ingest/image", json={"namespace": NAMESPACE, "items": image_batch})
    print(f"Image Batch Response: {res_img.status_code} - {res_img.text}")
    
    # 2. Ingest Text Batch
    print("Sending Text Batch...")
    res_text = requests.post(f"{VECTOR_URL}/ingest/text", json={"namespace": NAMESPACE, "items": text_batch})
    print(f"Text Batch Response: {res_text.status_code} - {res_text.text}")

    # 3. Perform a Hybrid Search
    query = data["queries"][0] 
    print(f"\nSearching for: '{query['text']}'...")
    
    search_res = requests.post(f"{VECTOR_URL}/search/hybrid", json={
        "namespace": NAMESPACE,
        "query_text": query["text"],
        "image_vector": query["vector"],
        "text_vector": query["vector"],
        "top_k": 3
    })
    
    print("\n--- Hybrid Search Results ---")
    results = search_res.json()
    print(f"Used Qwen (Hybrid Search): {results.get('used_qwen')}")
    for r in results.get("results", []):
        print(f"Match ID: {r['id']} - Final Score: {r['score']}")

if __name__ == "__main__":
    run_payload()
