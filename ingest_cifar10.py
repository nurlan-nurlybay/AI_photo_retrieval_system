import os
import requests
import torch
import torchvision
import torchvision.transforms as T
from transformers import CLIPProcessor, CLIPModel
import numpy as np
from io import BytesIO
from PIL import Image

SEAWEED_MASTER = "http://localhost:9333"
FAISS_URL = "http://localhost:8002/v1/vectors/add"

# load CLIP model
device = "cuda" if torch.cuda.is_available() else "cpu"
model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32").to(device)
processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

# CIFAR-10
transform = T.ToTensor()
dataset = torchvision.datasets.CIFAR10(root="./data", train=True, download=True, transform=transform)
classes = dataset.classes

def upload_to_seaweedfs(img: Image.Image) -> str:
    # ask master for fid
    assign = requests.get(f"{SEAWEED_MASTER}/dir/assign").json()
    fid = assign["fid"]
    url = assign["url"]

    # convert PIL image to bytes
    buf = BytesIO()
    img.save(buf, format="PNG")
    buf.seek(0)

    files = {"file": ("img.png", buf, "image/png")}
    resp = requests.post(f"http://{url}/{fid}", files=files)
    resp.raise_for_status()
    return f"http://{assign['publicUrl']}/{fid}"

def encode_image(img: Image.Image) -> list[float]:
    inputs = processor(images=img, return_tensors="pt").to(device)
    with torch.no_grad():
        vec = model.get_image_features(**inputs)
    arr = vec[0].cpu().numpy()
    arr = arr / np.linalg.norm(arr)  # L2 normalize
    return arr.tolist()

def insert_faiss(img_id: str, vector: list[float]):
    payload = {"id": img_id, "vector": vector}
    resp = requests.post(FAISS_URL, json=payload)
    resp.raise_for_status()

def main(n=100):
    for idx in range(n):
        img, label = dataset[idx]
        pil_img = T.ToPILImage()(img)
        class_name = classes[label]
        img_id = f"cifar_{idx}_{class_name}"

        # 1. upload
        url = upload_to_seaweedfs(pil_img)

        # 2. encode
        vec = encode_image(pil_img)

        # 3. insert into FAISS
        insert_faiss(img_id, vec)

        print(f"[{idx}] {class_name} → {url}")

if __name__ == "__main__":
    main(n=500)  # ingest first 500 CIFAR images
