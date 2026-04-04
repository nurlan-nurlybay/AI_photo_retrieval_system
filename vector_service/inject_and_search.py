import json
import requests

VECTOR_URL = "http://localhost:8006/v1"
JSON_FILE = "../test_running_containers/test_ml_service/ml_service_harvest.json"
NAMESPACE = "nurlan_gallery"

def run_payload():
    with open(JSON_FILE, 'r') as f:
        data = json.load(f)

    print(f"Loaded JSON. Injecting {len(data['gallery'])} items (Image + Text pairs)...")

    # 1. Ingest Data
    for item in data["gallery"]:
        # Ingest Image Vector
        res_img = requests.post(f"{VECTOR_URL}/ingest/image", json={
            "namespace": NAMESPACE,
            "id": item["id"],
            "image_vector": item["image_vector"]
        })
        
        # Ingest Text Vector & Tags
        res_text = requests.post(f"{VECTOR_URL}/ingest/text", json={
            "namespace": NAMESPACE,
            "id": item["id"],
            "text_vector": item["text_vector"],
            "tags": item["tags"]
        })
        
        print(f"ID {item['id']} Ingested -> Img: {res_img.status_code}, Txt: {res_text.status_code}")

    # 2. Perform a Hybrid Search
    query = data["queries"][0] 
    print(f"\nSearching for: '{query['text']}'...")
    
    search_res = requests.post(f"{VECTOR_URL}/search/hybrid", json={
        "namespace": NAMESPACE,
        "query_text": query["text"],
        "image_vector": query["vector"], # Send query embedding to image collection
        "text_vector": query["vector"],  # Send query embedding to text collection
        "top_k": 3
    })
    
    print("\n--- Hybrid Search Results ---")
    results = search_res.json()
    for r in results:
        print(f"Match ID: {r['id']} - Final Hybrid Score: {r['score']}")

if __name__ == "__main__":
    run_payload()

