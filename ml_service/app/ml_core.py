import os
import re
import json
import threading
import torch
import io
from PIL import Image
from typing import List, Dict, Any
from pillow_heif import register_heif_opener
from transformers import (
    AutoProcessor, 
    AutoModel, 
    Qwen3VLForConditionalGeneration, 
    BitsAndBytesConfig
)
from qwen_vl_utils import process_vision_info

register_heif_opener()

device = "cuda" if torch.cuda.is_available() else "cpu"

SIGLIP_ID = "google/siglip2-so400m-patch14-384"
QWEN_ID = "Qwen/Qwen3-VL-8B-Instruct"

# Global placeholders
siglip_processor: Any = None
siglip_model: Any = None
qwen_processor: Any = None
qwen_model: Any = None

# Thread locks to prevent concurrent model loading (double-checked locking pattern)
_siglip_lock = threading.Lock()
_qwen_lock = threading.Lock()

def warmup():
    """Bootstraps SigLIP only. Called by FastAPI lifespan."""
    global siglip_processor, siglip_model
    if siglip_model is None:
        with _siglip_lock:
            if siglip_model is None:
                print(f"--- Warming up SigLIP 2 ({SIGLIP_ID}) ---")
                siglip_processor = AutoProcessor.from_pretrained(SIGLIP_ID)
                siglip_model = AutoModel.from_pretrained(SIGLIP_ID, attn_implementation="sdpa").to(device)

def _load_qwen():
    """Lazy-loads Qwen3 only when the slow path is hit. Thread-safe."""
    global qwen_processor, qwen_model
    if qwen_model is None:
        with _qwen_lock:
            # Double-check after acquiring lock — another thread may have loaded it while we waited
            if qwen_model is None:
                print(f"--- Lazy-loading Qwen 3 ({QWEN_ID}) ---")
                quant_config = BitsAndBytesConfig(load_in_4bit=True, bnb_4bit_compute_dtype=torch.float16)
                qwen_processor = AutoProcessor.from_pretrained(QWEN_ID)
                qwen_model = Qwen3VLForConditionalGeneration.from_pretrained(
                    QWEN_ID, 
                    torch_dtype=torch.float16,
                    device_map={"": device}, 
                    quantization_config=quant_config, 
                    attn_implementation="sdpa"
                )

def _l2_norm(tensor_obj):
    """Safely extracts the tensor from HF wrappers and normalizes it."""
    if not isinstance(tensor_obj, torch.Tensor):
        if hasattr(tensor_obj, 'pooler_output') and tensor_obj.pooler_output is not None:
            tensor_obj = tensor_obj.pooler_output
        elif hasattr(tensor_obj, 'last_hidden_state') and tensor_obj.last_hidden_state is not None:
            tensor_obj = tensor_obj.last_hidden_state.mean(dim=1)
        else:
            tensor_obj = tensor_obj[0]
            
    return (tensor_obj / tensor_obj.norm(dim=-1, keepdim=True)).cpu().tolist()

def encode_text(texts: List[str]) -> List[List[float]]:
    if siglip_model is None: warmup()
    inputs = siglip_processor(text=texts, padding="max_length", truncation=True, return_tensors="pt").to(device)
    with torch.no_grad():
        feat = siglip_model.get_text_features(**inputs)
    return _l2_norm(feat)

def encode_image_fast(images_bytes: List[bytes]) -> List[List[float]]:
    if not images_bytes:
        return []
    if siglip_model is None: 
        warmup()
        
    # 1. Decode all images from RAM
    images = [Image.open(io.BytesIO(blob)).convert('RGB') for blob in images_bytes]
    
    # 2. Processor handles the stacking into a single batched tensor
    inputs = siglip_processor(images=images, return_tensors="pt").to(device)
    
    # 3. One forward pass for the entire array
    with torch.no_grad():
        features = siglip_model.get_image_features(**inputs)
        
    # 4. Matrix-wide L2 normalization
    features = features / features.norm(p=2, dim=-1, keepdim=True)
    
    return features.cpu().tolist()

def encode_image_slow(images_bytes: List[bytes]) -> List[Dict[str, Any]]:
    if not images_bytes:
        return []

    _load_qwen()
    assert qwen_model is not None and qwen_processor is not None
    
    # 1. Batch encode the images through SigLIP immediately
    img_vecs = encode_image_fast(images_bytes)
    
    messages_batch = []
    for blob in images_bytes:
        image = Image.open(io.BytesIO(blob)).convert('RGB')
        messages_batch.append([{"role": "user", "content": [
            {"type": "image", "image": image}, 
            {"type": "text", "text": 'Provide a description and 10 tags. Output ONLY a valid JSON object matching exactly this schema: {"description": "...", "tags": ["tag1", "tag2"]}.'}
        ]}])
        
    # 2. Apply chat template to the entire batch
    texts = [qwen_processor.apply_chat_template(m, tokenize=False, add_generation_prompt=True, enable_thinking=False) for m in messages_batch]
    
    # 3. Process vision info for the entire batch
    image_inputs, video_inputs = process_vision_info(messages_batch)  # type: ignore
    
    # CRITICAL: Left-padding is required for batched auto-regressive generation
    qwen_processor.tokenizer.padding_side = 'left'
    inputs = qwen_processor(text=texts, images=image_inputs, padding=True, return_tensors="pt").to(device)
    
    # 4. Generate descriptions for all images simultaneously
    with torch.inference_mode():
        output = qwen_model.generate(**inputs, max_new_tokens=600, do_sample=False)
        
    parsed_metadatas = []
    descriptions = []
    
    # 5. Decode and parse the batch outputs
    input_length = inputs["input_ids"].shape[-1]
    for i in range(len(images_bytes)):
        generated_ids = output[i][input_length:]
        decoded = qwen_processor.decode(generated_ids, skip_special_tokens=True)
        
        cleaned = re.sub(r'<think>.*?</think>', '', decoded, flags=re.DOTALL)
        match = re.search(r'\{.*\}', cleaned, re.DOTALL)
        metadata = json.loads(match.group(0)) if match else {"description": "error", "tags": []}
        
        parsed_metadatas.append(metadata)
        descriptions.append(metadata.get("description", "error"))

    # 6. Batch encode all the newly generated text descriptions through SigLIP
    txt_vecs = encode_text(descriptions)
    
    # 7. Assemble the final results
    results = []
    for i in range(len(images_bytes)):
        results.append({
            "description": parsed_metadatas[i].get("description", ""),
            "tags": parsed_metadatas[i].get("tags", []),
            "image_vector": img_vecs[i],
            "text_vector": txt_vecs[i]
        })
        
    return results
