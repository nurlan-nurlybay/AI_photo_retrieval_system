import json, os, torch
import numpy as np
from PIL import Image
from pillow_heif import register_heif_opener
from transformers import AutoProcessor, AutoModel

register_heif_opener()

img_out = "siglip_image_vectors.npy"
txt_out = "siglip_text_vectors.npy"

image_vectors = np.load(img_out, allow_pickle=True).item() if os.path.exists(img_out) else {}
text_vectors = np.load(txt_out, allow_pickle=True).item() if os.path.exists(txt_out) else {}

device = "cuda" if torch.cuda.is_available() else "cpu"

model_name = "google/siglip2-so400m-patch14-384"
print(f"Loading {model_name}...")

model = AutoModel.from_pretrained(model_name, attn_implementation="sdpa").to(device)
processor = AutoProcessor.from_pretrained(model_name)

def _l2_norm(tensor_obj):
    """Safely extracts the tensor from HF wrappers and normalizes it for numpy storage."""
    if not isinstance(tensor_obj, torch.Tensor):
        if hasattr(tensor_obj, 'pooler_output') and tensor_obj.pooler_output is not None:
            tensor_obj = tensor_obj.pooler_output
        elif hasattr(tensor_obj, 'last_hidden_state') and tensor_obj.last_hidden_state is not None:
            tensor_obj = tensor_obj.last_hidden_state.mean(dim=1)
        else:
            tensor_obj = tensor_obj[0]
            
    return (tensor_obj / tensor_obj.norm(dim=-1, keepdim=True)).cpu().numpy()

with open("metadata_prod_qwen.json", "r") as f:
    prod_meta = json.load(f)

image_dir = "./evaluation_dataset"

file_to_path = {}
for root, _, files in os.walk(image_dir):
    for f in files:
        if f.lower().endswith(('.png', '.jpg', '.jpeg', '.webp')):
            file_key = os.path.relpath(os.path.join(root, f), image_dir)
            file_to_path[file_key] = os.path.join(root, f)

count = 0
with torch.no_grad():
    for file, meta in prod_meta.items():
        if file in image_vectors and file in text_vectors:
            continue
        if file not in file_to_path:
            continue
            
        img_path = file_to_path[file]
        try:
            image = Image.open(img_path).convert('RGB')
            img_inputs = processor(images=image, return_tensors="pt").to(device)
            
            # 1. Extract and safely normalize image features
            img_feat = model.get_image_features(**img_inputs)
            image_vectors[file] = _l2_norm(img_feat)

            # 2. Extract and safely normalize text features
            text_inputs = processor(text=[meta.get('description', '')], padding="max_length", truncation=True, return_tensors="pt").to(device)
            txt_feat = model.get_text_features(**text_inputs)
            text_vectors[file] = _l2_norm(txt_feat)
            
            print(f"Vectorized {file}")
            count += 1
            if count % 50 == 0:
                np.save(img_out, image_vectors)
                np.save(txt_out, text_vectors)
                
        except Exception as e:
            print(f"Error vectorizing {file}: {e}")

np.save(img_out, image_vectors)
np.save(txt_out, text_vectors)
print("\n[SUCCESS] Vectorization complete.")

