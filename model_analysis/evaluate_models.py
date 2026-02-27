import os
import sys
import re
import torch
import json
import gc
from pathlib import Path
from PIL import Image, ImageFile
from tqdm import tqdm
from transformers import (
    AutoProcessor, 
    AutoModel, 
    BitsAndBytesConfig, 
    AutoModelForImageTextToText
)
from qwen_vl_utils import process_vision_info

# Environment & Config
os.environ["TRANSFORMERS_VERIFY_PICKLE"] = "FALSE"
os.environ["HF_HOME"] = "/workspace/huggingface_cache"
os.environ["PYTORCH_CUDA_ALLOC_CONF"] = "expandable_segments:True"
ImageFile.LOAD_TRUNCATED_IMAGES = True
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"

MASTER_FILES = ["nurlan_master_metadata.json", "elnaz_master_metadata.json"]
DATASET_ROOT = Path("./evaluation_dataset")
RELAXED_THRESHOLD = 0.95
RESULTS_FILE = "/workspace/benchmark_results.json"
VECTORS_FILE = "/workspace/siglip_embs.pt"
HITS_FILE = "/workspace/siglip_hits.json"
NAMES_FILE = "/workspace/valid_names.json"

bnb_config = BitsAndBytesConfig(
    load_in_8bit=True,
    llm_int8_enable_fp32_cpu_offload=True
)

def clear_vram():
    gc.collect()
    torch.cuda.empty_cache()
    torch.cuda.ipc_collect()

def load_dataset():
    all_metadata = {}
    paths = {}
    for master in MASTER_FILES:
        p = Path(master)
        if not p.exists(): continue
        with open(p, 'r') as f:
            data = json.load(f)
            person = master.split('_')[0]
            all_metadata.update(data)
            for img_name in data.keys():
                img_path = next(DATASET_ROOT.glob(f"{person}/**/{img_name}"), None)
                if img_path: paths[img_name] = img_path
    
    valid_names, valid_files = [], []
    print("🔍 Pre-verifying images to filter out corrupt files...")
    for name in tqdm(list(paths.keys()), desc="Verifying"):
        p = paths[name]
        try:
            with Image.open(p) as img:
                img.verify()
            with Image.open(p) as img:
                img.convert("RGB")
            valid_names.append(name)
            valid_files.append(p)
        except Exception:
            continue
            
    return all_metadata, valid_names, valid_files, paths

def save_results(model_name, metrics):
    results = {}
    if Path(RESULTS_FILE).exists():
        with open(RESULTS_FILE, "r") as f:
            try: results = json.load(f)
            except: results = {}
    results[model_name] = metrics
    with open(RESULTS_FILE, "w") as f:
        json.dump(results, f, indent=4)

def extract_features(outputs):
    if isinstance(outputs, torch.Tensor): return outputs
    if hasattr(outputs, "image_embeds") and outputs.image_embeds is not None: return outputs.image_embeds
    if hasattr(outputs, "text_embeds") and outputs.text_embeds is not None: return outputs.text_embeds
    if hasattr(outputs, "pooler_output") and outputs.pooler_output is not None: return outputs.pooler_output
    if hasattr(outputs, "last_hidden_state") and outputs.last_hidden_state is not None: return outputs.last_hidden_state[:, 0, :]
    return outputs

def run_bi_encoder(model_name, metadata, valid_names, valid_files, paths, is_siglip=False):
    print(f"\n⚙️ Loading Bi-Encoder: {model_name}...")
    model = AutoModel.from_pretrained(model_name, trust_remote_code=True).to(DEVICE)
    processor = AutoProcessor.from_pretrained(model_name, trust_remote_code=True)

    img_embs = []
    model.eval()
    with torch.no_grad():
        for i in tqdm(range(0, len(valid_files), 32), desc=f"Encoding Images ({model_name})"):
            batch = [Image.open(p).convert("RGB") for p in valid_files[i:i+32]]
            inputs = processor(images=batch, return_tensors="pt").to(DEVICE)
            outputs = model.get_image_features(**inputs) if hasattr(model, 'get_image_features') else model(**inputs)
            features = extract_features(outputs)
            img_embs.append(features / features.norm(p=2, dim=-1, keepdim=True))
    
    img_embs = torch.cat(img_embs)
    recall_1, recall_5, recall_10 = 0, 0, 0
    all_hits = []

    with torch.no_grad():
        for i, name in enumerate(tqdm(valid_names, desc=f"Running Queries ({model_name})")):
            query_text = (metadata[name]['description'] + " " + " ".join(metadata[name]['tags']))
            if is_siglip: query_text = query_text.lower()
            
            pad_style = "max_length" if is_siglip else "max_length"
            inputs = processor(text=[query_text], return_tensors="pt", padding=pad_style, max_length=64, truncation=True).to(DEVICE)
            
            t_out = model.get_text_features(**inputs) if hasattr(model, 'get_text_features') else model(**inputs)
            t_emb = extract_features(t_out)
            t_emb = t_emb / t_emb.norm(p=2, dim=-1, keepdim=True)
            
            scores = (t_emb @ img_embs.T).squeeze(0)
            top_k = torch.topk(scores, 10)
            indices = top_k.indices.tolist()
            
            if i in indices[:1]: recall_1 += 1
            if i in indices[:5]: recall_5 += 1
            if i in indices[:10]: recall_10 += 1
            
            all_hits.append([{"corpus_id": idx, "score": val} for idx, val in zip(indices, top_k.values.tolist())])

    save_results(model_name, {
        "Recall@1": round((recall_1/len(valid_names))*100, 2), 
        "Recall@5": round((recall_5/len(valid_names))*100, 2),
        "Recall@10": round((recall_10/len(valid_names))*100, 2)
    })
    
    if is_siglip:
        torch.save(img_embs, VECTORS_FILE)
        with open(HITS_FILE, "w") as f: json.dump(all_hits, f)
        with open(NAMES_FILE, "w") as f: json.dump(valid_names, f)

    del model, processor
    clear_vram()
    return all_hits, img_embs

def run_qwen(metadata, paths, size_label="8B"):
    if not (Path(VECTORS_FILE).exists() and Path(HITS_FILE).exists() and Path(NAMES_FILE).exists()):
        raise FileNotFoundError("Wave 3 requires Wave 2 to run first. Files missing.")
    
    img_embs = torch.load(VECTORS_FILE, map_location=DEVICE, weights_only=True)
    with open(HITS_FILE, "r") as f: siglip_hits = json.load(f)
    with open(NAMES_FILE, "r") as f: valid_names = json.load(f)

    checkpoint_file = f"/workspace/qwen_{size_label}_state.json"
    checkpoint = {"completed_idx": -1, "recall_1": 0, "recall_5": 0}
    if Path(checkpoint_file).exists():
        with open(checkpoint_file, "r") as f:
            checkpoint = json.load(f)
            print(f"♻️ Resuming Qwen {size_label} from index {checkpoint['completed_idx']}")

    print(f"\n⚙️ Loading Qwen3-VL-{size_label}-Instruct Reranker...")
    model_id = f"Qwen/Qwen3-VL-{size_label}-Instruct"
    processor = AutoProcessor.from_pretrained(model_id)
    
    Path("/workspace/offload").mkdir(exist_ok=True)
    
    max_mem = {0: "40GiB", "cpu": "100GiB"} if size_label == "8B" else {0: "30GiB", "cpu": "100GiB"}

    model = AutoModelForImageTextToText.from_pretrained(
        model_id, 
        quantization_config=bnb_config, 
        device_map="auto",
        max_memory=max_mem,
        attn_implementation="sdpa",
        offload_folder="/workspace/offload"
    )

    recall_1, recall_5 = checkpoint["recall_1"], checkpoint["recall_5"]

    for i, name in enumerate(tqdm(valid_names, desc=f"Reranking {size_label}", initial=checkpoint["completed_idx"]+1, total=len(valid_names))):
        if i <= checkpoint["completed_idx"]: continue
        
        query = metadata[name]['description'] + " " + " ".join(metadata[name]['tags'])
        candidates = siglip_hits[i]
        scores = []
        for hit in candidates:
            img_path = str(paths[valid_names[hit['corpus_id']]])
            msgs = [{"role": "user", "content": [{"type": "image", "image": img_path}, {"type": "text", "text": f"Score 0-100 for match: '{query}'. Num only."}]}]
            try:
                text = processor.apply_chat_template(msgs, tokenize=False, add_generation_prompt=True)
                image_inputs, *_ = process_vision_info(msgs)
                inputs = processor(text=[text], images=image_inputs, padding=True, return_tensors="pt").to(DEVICE)
                with torch.no_grad():
                    gen_ids = model.generate(**inputs, max_new_tokens=5)
                gen_text = processor.batch_decode(gen_ids[:, inputs.input_ids.shape[1]:], skip_special_tokens=True)[0]
                score_match = re.search(r'\d+', gen_text)
                score = int(score_match.group()) if score_match else 0
            except: score = 0
            scores.append((score, hit))
            
        scores.sort(key=lambda x: x[0], reverse=True)
        top_1_idx = scores[0][1]['corpus_id']
        top_5_indices = [s[1]['corpus_id'] for s in scores[:5]]
        
        if i == top_1_idx or torch.nn.functional.cosine_similarity(img_embs[i].unsqueeze(0), img_embs[top_1_idx].unsqueeze(0)).item() >= RELAXED_THRESHOLD:
            recall_1 += 1
        if i in top_5_indices or any(torch.nn.functional.cosine_similarity(img_embs[i].unsqueeze(0), img_embs[idx].unsqueeze(0)).item() >= RELAXED_THRESHOLD for idx in top_5_indices):
            recall_5 += 1

        if i % 5 == 0:
            with open(checkpoint_file, "w") as f:
                json.dump({"completed_idx": i, "recall_1": recall_1, "recall_5": recall_5}, f)

    save_results(f"Qwen3-VL-{size_label}-Final", {"Recall@1": round((recall_1/len(valid_names))*100, 2), "Recall@5": round((recall_5/len(valid_names))*100, 2)})

if __name__ == "__main__":
    args = sys.argv[1:]
    if not args:
        print("Usage: python3 evaluate_models.py <waves> [size]\nExample: python3 evaluate_models.py 1 2 3 8B")
        sys.exit(1)
    
    waves = [int(arg) for arg in args if arg.isdigit() and int(arg) in [1, 2, 3]]
    
    size_label = "8B"
    if "32B" in args or "32" in args:
        size_label = "32B"
    
    if not waves:
        print("Error: Specify at least one valid wave (1, 2, or 3).")
        sys.exit(1)

    meta, v_names, v_files, all_paths = load_dataset()
    
    if 1 in waves:
        run_bi_encoder("laion/CLIP-ViT-bigG-14-laion2B-39B-b160k", meta, v_names, v_files, all_paths, is_siglip=False)
    if 2 in waves:
        run_bi_encoder("google/siglip2-giant-opt-patch16-384", meta, v_names, v_files, all_paths, is_siglip=True)
    if 3 in waves:
        run_qwen(meta, all_paths, size_label)
