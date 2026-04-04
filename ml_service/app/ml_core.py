import json
import torch
import numpy as np
import structlog
from typing import List, Dict, Any, Optional
from PIL import Image
from io import BytesIO
from transformers import (
    AutoProcessor, 
    AutoModel, 
    Qwen2VLForConditionalGeneration
)
from qwen_vl_utils import process_vision_info

logger = structlog.get_logger(__name__)

DEVICE = "cuda" if torch.cuda.is_available() else "cpu"
torch.set_float32_matmul_precision('high')

SIGLIP_MODEL_ID = "google/siglip-2-so400m-384"
QWEN_MODEL_ID = "Qwen/Qwen3-VL-8B-Instruct"

# Type as Any to stop strict linters from complaining about HF dynamic factory methods
_siglip: Any = None
_siglip_proc: Any = None
_qwen: Any = None
_qwen_proc: Any = None

def warmup() -> None:
    global _siglip, _siglip_proc, _qwen, _qwen_proc
    log = logger.bind(device=DEVICE)
    try:
        log.info("model_load_start", model=SIGLIP_MODEL_ID)
        _siglip = AutoModel.from_pretrained(SIGLIP_MODEL_ID).to(DEVICE).eval()
        _siglip_proc = AutoProcessor.from_pretrained(SIGLIP_MODEL_ID)

        log.info("model_load_start", model=QWEN_MODEL_ID, quantization="4-bit")
        _qwen = Qwen2VLForConditionalGeneration.from_pretrained(
            QWEN_MODEL_ID, 
            torch_dtype=torch.bfloat16, 
            device_map="auto", 
            load_in_4bit=True
        ).eval()
        _qwen_proc = AutoProcessor.from_pretrained(QWEN_MODEL_ID)
        log.info("warmup_complete")
    except Exception as e:
        log.error("warmup_failed", error=str(e), exc_info=True)
        raise

def _l2_norm(vecs: torch.Tensor) -> List[List[float]]:
    normed = torch.nn.functional.normalize(vecs, p=2, dim=-1)
    return normed.cpu().detach().numpy().tolist()

def encode_text(texts: List[str]) -> List[List[float]]:
    if _siglip is None or _siglip_proc is None:
        raise RuntimeError("MLCore: SigLIP not initialized.")
    
    inputs = _siglip_proc(text=texts, padding="max_length", truncation=True, return_tensors="pt").to(DEVICE)
    with torch.inference_mode():
        features = _siglip.get_text_features(**inputs)
    return _l2_norm(features)

def encode_image_fast(images: List[bytes]) -> List[List[float]]:
    if _siglip is None or _siglip_proc is None:
        raise RuntimeError("MLCore: SigLIP not initialized.")
        
    pil_list = [Image.open(BytesIO(b)).convert("RGB") for b in images]
    inputs = _siglip_proc(images=pil_list, padding=True, return_tensors="pt").to(DEVICE)
    with torch.inference_mode():
        features = _siglip.get_image_features(**inputs)
    return _l2_norm(features)

def encode_image_slow(images: List[bytes]) -> List[Dict[str, Any]]:
    if _qwen is None or _qwen_proc is None:
        raise RuntimeError("MLCore: Qwen3-VL not initialized.")

    results = []
    pil_images = [Image.open(BytesIO(b)).convert("RGB") for b in images]
    
    for i, img in enumerate(pil_images):
        log = logger.bind(batch_index=i, resolution=img.size)
        try:
            messages = [{"role": "user", "content": [
                {"type": "image", "image": img},
                {"type": "text", "text": "Describe this image and list 10 tags. Format as JSON: {'description': '...', 'tags': []}"}
            ]}]
            
            text = _qwen_proc.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
            
            # FIXED: Expecting 2 values instead of 3
            image_inputs, video_inputs, *rest = process_vision_info(messages) 
            
            inputs = _qwen_proc(text=[text], images=image_inputs, padding=True, return_tensors="pt").to(DEVICE)
            
            with torch.inference_mode(), torch.autocast("cuda", dtype=torch.bfloat16):
                output = _qwen.generate(**inputs, max_new_tokens=400)
            
            decoded = _qwen_proc.batch_decode(output, skip_special_tokens=True)[0]
            clean_json = decoded.replace("```json", "").replace("```", "").strip()
            
            parsed = json.loads(clean_json)
            
            semantic_payload = f"{parsed.get('description')} {' '.join(parsed.get('tags', []))}"
            vec = encode_text([semantic_payload])[0]
            
            results.append({**parsed, "text_vector": vec})
            log.info("image_processed")
            
        except Exception as e:
            log.error("processing_error", error=str(e))
            results.append({"description": "error", "tags": [], "text_vector": [0.0]*1152})
            
    return results
