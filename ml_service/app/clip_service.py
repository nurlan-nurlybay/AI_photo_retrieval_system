import torch
import numpy as np
from transformers import CLIPProcessor, CLIPModel
from PIL import Image
from io import BytesIO

device = "cuda" if torch.cuda.is_available() else "cpu"
model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32").to(device)
processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

def encode_text(text: str) -> list[float]:
    inputs = processor(text=[text], return_tensors="pt", padding=True).to(device)
    with torch.no_grad():
        vec = model.get_text_features(**inputs)
    arr = vec[0].cpu().numpy()

    # L2 normalize
    arr = arr / np.linalg.norm(arr)
    return arr.tolist()

def encode_image(data: bytes) -> list[float]:
    img = Image.open(BytesIO(data)).convert("RGB")
    inputs = processor(images=img, return_tensors="pt").to(device)
    with torch.no_grad():
        vec = model.get_image_features(**inputs)
    arr = vec[0].cpu().numpy()

    # L2 normalize
    arr = arr / np.linalg.norm(arr)
    return arr.tolist()
