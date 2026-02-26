import os
import json
import shutil
import base64
import time
import io
from pathlib import Path
from openai import OpenAI
from tqdm import tqdm
from dotenv import load_dotenv
from concurrent.futures import ThreadPoolExecutor
from PIL import Image
import pillow_heif  # ! pip install pillow-heif

# Register HEIF support with Pillow
pillow_heif.register_heif_opener()

load_dotenv()
API_KEY = os.getenv("API_KEY")

BASE_DIR = Path(__file__).parent
SRC_DIR = BASE_DIR / "elnaz_images"
PERSON_NAME = "elnaz" 
EVAL_DIR = BASE_DIR / "evaluation_dataset" / PERSON_NAME
BASE_URL = "https://dashscope-intl.aliyuncs.com/compatible-mode/v1"
MODEL_ID = "qwen-vl-plus" 

client = OpenAI(api_key=API_KEY, base_url=BASE_URL, timeout=60.0)

def encode_and_preprocess_image(image_path: Path, max_size_mb=9.5):
    file_size_mb = image_path.stat().st_size / (1024 * 1024)
    suffix = image_path.suffix.lower()
    
    # Force process if it's HEIC or too large
    if file_size_mb > max_size_mb or suffix in [".heic", ".heif"] or max_size_mb > 2048:
        with Image.open(image_path) as img:
            if img.mode != "RGB":
                img = img.convert("RGB")
            
            img.thumbnail((1560, 1560), Image.Resampling.LANCZOS)
            buffer = io.BytesIO()
            img.save(buffer, format="JPEG", quality=85, optimize=True)
            return base64.b64encode(buffer.getvalue()).decode('utf-8')
    
    with open(image_path, "rb") as image_file:
        return base64.b64encode(image_file.read()).decode('utf-8')

def get_ai_label(image_path):
    try:
        base64_image = encode_and_preprocess_image(image_path)
    except Exception as e:
        print(f"❌ Read error on {image_path.name}: {e}")
        return {"description": "error", "tags": []}

    prompt = (
        "Analyze this image and output a JSON object with: "
        "1. 'description': A 1-sentence technical summary of the scene. "
        "2. 'tags': A list of matching categories. "
        "Format: {'description': '...', 'tags': ['tag1', 'tag2', ...]}"
    )

    for attempt in range(3):
        try:
            response = client.chat.completions.create(
                model=MODEL_ID,
                messages=[{"role": "user", "content": [
                    {"type": "text", "text": prompt},
                    {"type": "image_url", "image_url": {"url": f"data:image/jpeg;base64,{base64_image}"}}
                ]}],
                response_format={"type": "json_object"}
            )
            content = response.choices[0].message.content
            return json.loads(content) if content else {"description": "error", "tags": []}
        except Exception as e:
            if attempt == 2: return {"description": "error", "tags": []}
            time.sleep(5)
    return {"description": "error", "tags": []}

def process_subfolder(sub):
    metadata_file = sub / "metadata.json"
    if metadata_file.exists(): return 
    
    sub_metadata = {}
    # Added .heic to the glob
    sub_imgs = sorted([p for p in sub.glob("*") if p.suffix.lower() in [".jpg", ".png", ".jpeg", ".heic"]])
    
    for img in sub_imgs:
        sub_metadata[img.name] = get_ai_label(img)
    
    with open(metadata_file, "w") as f:
        json.dump(sub_metadata, f, indent=4)

def process_images():
    # Added .heic to the source scan
    images = sorted([p for p in SRC_DIR.glob("*") if p.suffix.lower() in [".jpg", ".png", ".jpeg", ".heic"]])
    if not images:
        print(f"❌ No images found in {SRC_DIR}")
        return
        
    print(f"🚀 Processing {len(images)} images for {PERSON_NAME}...")

    for i, img_path in enumerate(images):
        batch_num = (i // 50) + 1
        sub_num = ((i % 50) // 10) + 1
        target_dir = EVAL_DIR / f"batch_{batch_num:02d}" / f"sub_{sub_num:02d}"
        target_dir.mkdir(parents=True, exist_ok=True)
        
        # Convert HEIC to JPG for the evaluation folder so SigLIP has no issues later
        if img_path.suffix.lower() in [".heic", ".heif"]:
            new_img_path = target_dir / (img_path.stem + ".jpg")
        else:
            new_img_path = target_dir / img_path.name
            
        if not new_img_path.exists():
            shutil.copy(img_path, new_img_path)

    subfolders = sorted(list(EVAL_DIR.glob("batch_*/sub_*")))
    pending_subs = [s for s in subfolders if not (s / "metadata.json").exists()]
    
    with ThreadPoolExecutor(max_workers=5) as executor:
        list(tqdm(executor.map(process_subfolder, pending_subs), total=len(pending_subs), desc="Overall Progress"))

if __name__ == "__main__":
    process_images()
