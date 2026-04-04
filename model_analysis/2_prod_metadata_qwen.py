import json, os, torch
from transformers import Qwen2VLForConditionalGeneration, AutoProcessor
from qwen_vl_utils import process_vision_info

output_file = "metadata_prod_qwen.json"
model_id = "Qwen/Qwen3-VL-8B-Instruct"

if os.path.exists(output_file):
    with open(output_file, "r") as f: prod_metadata = json.load(f)
    print(f"Resuming from {len(prod_metadata)} records...")
else: 
    prod_metadata = {}

print(f"Loading {model_id}...")
model = Qwen2VLForConditionalGeneration.from_pretrained(
    model_id, torch_dtype=torch.float16, device_map="auto", load_in_4bit=True, attn_implementation="sdpa"
)
processor = AutoProcessor.from_pretrained(model_id)

image_dir = "./evaluation_dataset"
all_files = []
for root, _, files in os.walk(image_dir):
    for f in files:
        if f.lower().endswith(('.png', '.jpg', '.jpeg')):
            all_files.append(os.path.join(root, f))
all_files.sort(key=lambda x: os.path.basename(x))

for img_path in all_files:
    file = os.path.basename(img_path)
    if file in prod_metadata: continue
    try:
        messages = [{"role": "user", "content": [{"type": "image", "image": img_path}, {"type": "text", "text": "Describe this image and list 10 tags. Format as JSON: {'description': '...', 'tags': []}"}]}]
        text = processor.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
        image_inputs, _ = process_vision_info(messages)
        inputs = processor(text=[text], images=image_inputs, padding=True, return_tensors="pt").to("cuda")
        
        with torch.inference_mode():
            output = model.generate(**inputs, max_new_tokens=300)
            
        decoded = processor.batch_decode(output, skip_special_tokens=True)[0]
        prod_metadata[file] = json.loads(decoded.replace("```json", "").replace("```", "").strip())
        
        with open(output_file, "w") as f: json.dump(prod_metadata, f, indent=4)
        print(f"Processed Prod for {file}")
        del inputs, output; torch.cuda.empty_cache()
    except Exception as e: 
        print(f"Error on {file}: {e}")
