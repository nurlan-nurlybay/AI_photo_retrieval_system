import json, os, torch
import numpy as np
from PIL import Image
from transformers import AutoProcessor, AutoModel

img_out = "siglip_image_vectors.npy"
txt_out = "siglip_text_vectors.npy"

image_vectors = np.load(img_out, allow_pickle=True).item() if os.path.exists(img_out) else {}
text_vectors = np.load(txt_out, allow_pickle=True).item() if os.path.exists(txt_out) else {}

device = "cuda" if torch.cuda.is_available() else "cpu"
model_name = "google/siglip-2-so400m-384"
model = AutoModel.from_pretrained(model_name).to(device)
processor = AutoProcessor.from_pretrained(model_name)

with open("metadata_prod_qwen.json", "r") as f:
    prod_meta = json.load(f)

image_dir = "./evaluation_dataset"

# Build path lookup to handle nested directories safely
file_to_path = {}
for root, _, files in os.walk(image_dir):
    for f in files:
        if f.lower().endswith(('.png', '.jpg', '.jpeg')):
            file_to_path[f] = os.path.join(root, f)

count = 0
with torch.no_grad():
    for file, meta in prod_meta.items():
        if file in image_vectors and file in text_vectors:
            continue
            
        if file not in file_to_path:
            continue
            
        img_path = file_to_path[file]
        
        # 1. Image Vector
        image = Image.open(img_path).convert('RGB')
        img_inputs = processor(images=image, return_tensors="pt").to(device)
        img_feat = model.get_image_features(**img_inputs)
        image_vectors[file] = (img_feat / img_feat.norm(dim=-1, keepdim=True)).cpu().numpy()

        # 2. Text Vector
        text_inputs = processor(text=[meta['description']], padding="max_length", return_tensors="pt").to(device)
        txt_feat = model.get_text_features(**text_inputs)
        text_vectors[file] = (txt_feat / txt_feat.norm(dim=-1, keepdim=True)).cpu().numpy()
        
        print(f"Vectorized {file}")
        count += 1
        
        if count % 50 == 0:
            np.save(img_out, image_vectors)
            np.save(txt_out, text_vectors)
            print(f"--- Checkpoint saved at {len(image_vectors)} vectors ---")

np.save(img_out, image_vectors)
np.save(txt_out, text_vectors)

