from transformers import CLIPProcessor, CLIPModel
import torch
import numpy as np

model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32")
processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

def encode_text(text: str) -> list[float]:
    inputs = processor(text=[text], return_tensors="pt", padding=True)
    with torch.no_grad():
        vec = model.get_text_features(**inputs)
    arr = vec[0].cpu().numpy()

    # L2 normalize
    arr = arr / np.linalg.norm(arr)
    return arr.tolist()
