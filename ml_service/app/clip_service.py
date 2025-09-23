from transformers import CLIPProcessor, CLIPModel
import torch

model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32")
processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

def encode_text(text: str) -> list[float]:
    inputs = processor(text=[text], return_tensors="pt", padding=True)
    with torch.no_grad():
        vec = model.get_text_features(**inputs)
    return vec[0].cpu().numpy().tolist()
