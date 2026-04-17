import json, os, torch, re
from PIL import Image
from pillow_heif import register_heif_opener
from transformers import Qwen3VLForConditionalGeneration, AutoProcessor, BitsAndBytesConfig
from qwen_vl_utils import process_vision_info

register_heif_opener()

output_file = "metadata_prod_qwen.json"
model_id = "Qwen/Qwen3-VL-8B-Instruct"

if os.path.exists(output_file):
    with open(output_file, "r") as f: prod_metadata = json.load(f)
else: 
    prod_metadata = {}

print(f"Loading {model_id}...")
quantization_config = BitsAndBytesConfig(load_in_4bit=True, bnb_4bit_compute_dtype=torch.float16)

model = Qwen3VLForConditionalGeneration.from_pretrained(
    model_id, torch_dtype=torch.float16, device_map="auto", quantization_config=quantization_config, attn_implementation="sdpa"
)
processor = AutoProcessor.from_pretrained(model_id)

image_dir = "./evaluation_dataset"
all_files = []
for root, _, files in os.walk(image_dir):
    for f in files:
        if f.lower().endswith(('.png', '.jpg', '.jpeg', '.webp')):
            all_files.append(os.path.join(root, f))
all_files.sort()

for img_path in all_files:
    file_key = os.path.relpath(img_path, image_dir)
    if file_key in prod_metadata: continue
    
    try:
        _ = Image.open(img_path).convert("RGB")
    except Exception:
        continue

    try:
        messages = [{"role": "user", "content": [{"type": "image", "image": img_path}, {"type": "text", "text": 'Provide a description and 10 tags. Output ONLY a valid JSON object matching exactly this schema: {"description": "...", "tags": ["tag1", "tag2", ...]}. Do not include any other text.'}]}]
        
        text = processor.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
        image_inputs, video_inputs, *rest = process_vision_info(messages)
        
        inputs = processor(text=[text], images=image_inputs, return_tensors="pt").to("cuda")
        
        with torch.inference_mode():
            output = model.generate(**inputs, max_new_tokens=600, do_sample=False)  # type: ignore
            
        # CRITICAL FIX: Slice off the input prompt tokens so we ONLY decode the new response
        generated_ids = output[0][inputs["input_ids"].shape[-1]:]
        decoded = processor.decode(generated_ids, skip_special_tokens=True)
        
        cleaned_text = re.sub(r'<think>.*?</think>', '', decoded, flags=re.DOTALL)
        
        start_idx = cleaned_text.find('{')
        end_idx = cleaned_text.rfind('}')
        
        if start_idx != -1 and end_idx != -1:
            json_str = cleaned_text[start_idx:end_idx+1]
            prod_metadata[file_key] = json.loads(json_str)
            
            with open(output_file, "w") as f: json.dump(prod_metadata, f, indent=4)
            print(f"Processed Prod for {file_key}")
        else:
            print(f"No JSON brackets found for {file_key}")
            
        del inputs, output; torch.cuda.empty_cache()
        
    except json.JSONDecodeError as e:
        print(f"\nJSON Error on {file_key}: {e}\n--- CLEANED OUTPUT ---\n{cleaned_text}\n-------------------\n")  # type: ignore
    except Exception as e: 
        print(f"Error on {file_key}: {e}")
