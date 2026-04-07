import json, os, sys, torch, re
from PIL import Image
from pillow_heif import register_heif_opener
from transformers import AutoProcessor, Gemma3ForConditionalGeneration, BitsAndBytesConfig

# Tell Pillow how to read Apple HEIF/HEIC files
register_heif_opener()

output_file = "metadata_gt_gemma.json"
model_id = "google/gemma-3-27b-it"

if os.path.exists(output_file):
    with open(output_file, "r") as f: ground_truth = json.load(f)
    print(f"Resuming from {len(ground_truth)} records...")
else: 
    ground_truth = {}

print(f"Loading {model_id} (4-bit)...")
quantization_config = BitsAndBytesConfig(load_in_4bit=True, bnb_4bit_compute_dtype=torch.bfloat16)

model = Gemma3ForConditionalGeneration.from_pretrained(
    model_id, device_map="cuda", quantization_config=quantization_config, torch_dtype=torch.bfloat16, attn_implementation="sdpa"
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
    
    if file_key in ground_truth: continue
    
    try:
        image = Image.open(img_path).convert("RGB")
    except Exception as e:
        print(f"Skipping corrupted image {file_key}: {e}")
        continue

    try:
        messages = [{"role": "user", "content": [{"type": "image"}, {"type": "text", "text": "Provide a description and 10 tags. Output ONLY a valid JSON object matching exactly this schema: {'description': '...', 'tags': ['tag1', 'tag2', ...]}. Do not include any other text."}]}]
        
        prompt_text = processor.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
        inputs = processor(text=prompt_text, images=image, return_tensors="pt").to(model.device)
        
        with torch.inference_mode():
            output = model.generate(**inputs, max_new_tokens=500, do_sample=False)
            
        decoded = processor.decode(output[0][inputs["input_ids"].shape[-1]:], skip_special_tokens=True)
        
        json_match = re.search(r'\{.*\}', decoded, re.DOTALL)
        if json_match:
            clean_json = json_match.group(0)
            ground_truth[file_key] = json.loads(clean_json)
        else:
            clean_json = decoded.replace("```json", "").replace("```", "").strip()
            ground_truth[file_key] = json.loads(clean_json)
        
        with open(output_file, "w") as f: json.dump(ground_truth, f, indent=4)
        print(f"Processed GT for {file_key}")
        del inputs, output, image; torch.cuda.empty_cache()
        
    except json.JSONDecodeError:
        print(f"JSON Error on {file_key}. Model generated:\n{decoded}\n")
    except Exception as e: 
        print(f"Error on {file_key}: {e}")
