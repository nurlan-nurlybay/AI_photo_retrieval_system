import requests
import json
import os

ML_URL = "http://localhost:8005/v1"
IMAGE_DIR = "images"
OUTPUT_FILE = "ml_service_harvest.json"

# Mapping the filenames to your descriptions
QUERY_STRINGS = [
    "Lipstick", "Black glasses with green case", "Black backpack", 
    "Laptop charger", "A desk with a laptop and a bunch of empty bottles", 
    "A desk with random food waste", "A clean desk with a black laptop in a study room", 
    "Study room", "Happy selfie of a young man", "Selfie a tired young man", 
    "Blue chair", "Hand", "Slippers"
]

def harvest():
    harvested_data = {
        "queries": [],
        "gallery": []
    }

    # 1. HARVEST TEXT VECTORS (For the 13 queries)
    print("--- Harvesting Text Vectors ---")
    text_res = requests.post(f"{ML_URL}/encode/text/", json={"texts": QUERY_STRINGS}).json()
    for i, text in enumerate(QUERY_STRINGS):
        harvested_data["queries"].append({
            "text": text,
            "vector": text_res["vectors"][i]
        })

    # 2. HARVEST IMAGE DATA (Fast path only, mocking the slow path)
    print("--- Harvesting Image Intelligence (Skipping Qwen, mocking text data) ---")
    for i in range(1, 14):
        file_name = f"{i}.jpg"
        file_path = os.path.join(IMAGE_DIR, file_name)
        
        if not os.path.exists(file_path):
            print(f"Skipping {file_name} - not found.")
            continue

        print(f"Processing {file_name}...")
        
        # A. Get the real SigLIP image vector
        with open(file_path, "rb") as f:
            files = {"files": (file_name, f, "image/jpeg")}
            fast_res = requests.post(f"{ML_URL}/encode/image/fast/", files=files).json()
            
        # B. Mock the Qwen outputs using your query strings
        mock_desc = QUERY_STRINGS[i-1]
        # Just use the words in the description as the tags
        mock_tags = mock_desc.lower().split() 
        
        # C. Get a real 1152-dim text vector for the mock description
        text_req = requests.post(f"{ML_URL}/encode/text/", json={"texts": [mock_desc]}).json()
        
        harvested_data["gallery"].append({
            "id": i,
            "file_name": file_name,
            "image_vector": fast_res["vectors"][0],
            "description": mock_desc,
            "tags": mock_tags,
            "text_vector": text_req["vectors"][0]
        })

    # 3. SAVE TO JSON
    with open(OUTPUT_FILE, "w") as f:
        json.dump(harvested_data, f, indent=4)
    
    print(f"\n[SUCCESS] Harvest complete. Data saved to {OUTPUT_FILE}")

if __name__ == "__main__":
    harvest()

