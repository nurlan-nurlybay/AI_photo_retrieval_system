#!/usr/bin/env python3
import os
import json
import time
from pathlib import Path

import torch
import numpy as np
from PIL import Image
from tqdm import tqdm

from .utils import list_images, cosine_sim, compute_f1_at_threshold, mrr
from .models_clip import OpenCLIPEncoder, HFCLIPWithPEFT

# Config
DATA_DIR = Path(os.environ.get("EVAL_DATA_DIR", "./eval_data"))
RESULTS_DIR = Path(os.environ.get("EVAL_RESULTS_DIR", "./eval/results"))
RESULTS_DIR.mkdir(parents=True, exist_ok=True)

USE_QWEN = os.environ.get("ENABLE_QWEN", "1") == "1"
QWEN_MODEL_ID = os.environ.get("QWEN_VL_MODEL", "Qwen/Qwen2-VL-2B-Instruct")
TOPK_FOR_RERANK = int(os.environ.get("TOPK_FOR_RERANK", "50"))
ADAPTER_DIR = Path(os.environ.get("CLIP_ADAPTER_DIR", "./fine_tuning/clip_finetuned_epoch_3"))


def build_labels(categories: list[str], img_labels: list[str]) -> np.ndarray:
    # Rows = queries (categories), Cols = images; 1 if an image belongs to that category
    y = np.zeros((len(categories), len(img_labels)), dtype=np.int32)
    cat_to_idx = {c: i for i, c in enumerate(categories)}
    for j, lab in enumerate(img_labels):
        i = cat_to_idx.get(lab)
        if i is not None:
            y[i, j] = 1
    return y


def sweep_thresholds(sim: np.ndarray, y_true: np.ndarray) -> dict[str, float]:
    best_t, best_f1 = 0.0, 0.0
    for t in np.linspace(0.0, 1.0, 101):
        f1 = compute_f1_at_threshold(sim, y_true, float(t))
        if f1 > best_f1:
            best_f1, best_t = f1, float(t)
    return {"optimal_threshold": best_t, "max_f1": best_f1}


def evaluate_model(name: str, img_emb: np.ndarray, txt_emb: np.ndarray, y_true: np.ndarray) -> dict[str, float]:
    sim = cosine_sim(txt_emb, img_emb)
    swe = sweep_thresholds(sim, y_true)
    swe["mrr"] = mrr(sim, y_true)
    return swe


def rerank_with_qwen(siglip_sim: np.ndarray, image_paths: list[Path], categories: list[str]) -> np.ndarray:
    # For each query, select TopK indices from SigLIP sim and rerank them via Qwen.
    from .qwen_reranker import QwenReranker
    reranker = QwenReranker(QWEN_MODEL_ID)

    # Optional: restrict reranking to a subset of queries for quick validation
    only_queries = os.environ.get("QWEN_QUICK_QUERIES", "").strip()
    subset = None
    if only_queries:
        subset = [q.strip() for q in only_queries.split(",") if q.strip()]

    num_q, num_img = siglip_sim.shape
    new_scores = np.copy(siglip_sim)

    queries_to_process = []
    for qi in range(num_q):
        if subset is not None and categories[qi] not in subset:
            continue
        queries_to_process.append(qi)

    for qi in tqdm(queries_to_process, desc="Qwen reranking queries", unit="query"):
        topk_idx = np.argsort(-siglip_sim[qi])[:TOPK_FOR_RERANK]
        imgs = [Image.open(image_paths[j]).convert("RGB") for j in topk_idx]
        scores = reranker.batch_score(imgs, categories[qi])
        for local, j in enumerate(topk_idx):
            new_scores[qi, j] = siglip_sim[qi, j] * float(scores[local])
    return new_scores


def main():
    print(f"\n{'='*60}")
    print("BENCHMARK: Vision-Retrieval Pipeline Evaluation")
    print(f"{'='*60}")
    print(f"Dataset directory : {DATA_DIR.resolve()}")
    print(f"Results directory : {RESULTS_DIR.resolve()}")
    print(f"Qwen reranker    : {'enabled' if USE_QWEN else 'disabled'}")
    if USE_QWEN:
        print(f"  Model           : {QWEN_MODEL_ID}")
        print(f"  Top-K           : {TOPK_FOR_RERANK}")
    print(f"{'='*60}\n")

    print("Loading dataset...")
    image_paths, img_labels = list_images(DATA_DIR)
    if len(image_paths) == 0:
        print("No images found under eval_data/. Did you run collect_unsplash.py?")
        return
    categories = sorted(list(set(img_labels)))
    print(f"Found {len(image_paths)} images in {len(categories)} categories")

    # Build labels matrix
    y_true = build_labels(categories, img_labels)

    # Load images to memory (might be heavy for full set; OK for quick run)
    print(f"Loading {len(image_paths)} images into memory...")
    images = [Image.open(p).convert("RGB") for p in tqdm(image_paths, desc="Loading images")]

    results: dict[str, dict[str, float]] = {}
    total_start = time.time()
    n_models = 3 + (1 if USE_QWEN else 0)

    # Prompt templates optimised for each model's training distribution
    texts_clip = [f"a photo of {cat}" for cat in categories]
    texts_ft = [f"A photo of a {cat.replace('_', ' ')}" for cat in categories]
    texts_siglip = [f"an image depicting {cat}" for cat in categories]

    # 1) Baseline CLIP ViT-L/14 (open_clip, openai weights)
    print(f"\n{'='*60}")
    print(f"[1/{n_models}] Encoding with Baseline CLIP ViT-L/14 (openai)")
    print(f"{'='*60}")
    t0 = time.time()
    enc_clip_l14 = OpenCLIPEncoder("ViT-L-14", pretrained="openai")
    img_clip_l14 = enc_clip_l14.encode_images(images)
    txt_clip_l14 = enc_clip_l14.encode_texts(texts_clip)
    res_clip_l14 = evaluate_model("CLIP ViT-L/14", img_clip_l14, txt_clip_l14, y_true)
    results["CLIP_ViT_L_14"] = res_clip_l14
    print(f"  Done in {time.time()-t0:.1f}s  |  F1={res_clip_l14['max_f1']:.4f}  MRR={res_clip_l14['mrr']:.4f}")

    # 2) Fine-tuned CLIP (HF + PEFT adapters)
    print(f"\n{'='*60}")
    print(f"[2/{n_models}] Encoding with Fine-tuned CLIP (ViT-B/32 + LoRA)")
    print(f"{'='*60}")
    t0 = time.time()
    enc_ft = HFCLIPWithPEFT(base_model_name="openai/clip-vit-base-patch32", adapter_dir=ADAPTER_DIR)
    img_ft = enc_ft.encode_images(images)
    txt_ft = enc_ft.encode_texts(texts_ft)
    res_ft = evaluate_model("FineTuned CLIP ViT-B/32", img_ft, txt_ft, y_true)
    results["CLIP_ViT_B_32_FT"] = res_ft
    print(f"  Done in {time.time()-t0:.1f}s  |  F1={res_ft['max_f1']:.4f}  MRR={res_ft['mrr']:.4f}")

    # 3) SigLIP 2 (configurable) via open_clip (hf-hub)
    siglip_id = os.environ.get("SIGLIP_MODEL_ID", "ViT-L-16-SigLIP2-384")
    siglip_pretrained = os.environ.get("SIGLIP_PRETRAINED", "webli")
    siglip_label = os.environ.get("SIGLIP_LABEL", siglip_id.replace("-", "_"))
    print(f"\n{'='*60}")
    print(f"[3/{n_models}] Encoding with SigLIP 2: {siglip_id} ({siglip_pretrained})")
    print(f"{'='*60}")
    t0 = time.time()
    enc_sig = OpenCLIPEncoder(siglip_id, pretrained=siglip_pretrained)
    img_sig = enc_sig.encode_images(images)
    txt_sig = enc_sig.encode_texts(texts_siglip)
    res_sig = evaluate_model(siglip_label, img_sig, txt_sig, y_true)
    results[siglip_label] = res_sig
    print(f"  Done in {time.time()-t0:.1f}s  |  F1={res_sig['max_f1']:.4f}  MRR={res_sig['mrr']:.4f}")

    # Free GPU memory from embedding models before loading Qwen
    del enc_clip_l14, enc_ft, enc_sig
    import gc; gc.collect()
    if torch.cuda.is_available():
        torch.cuda.empty_cache()

    # 4) Stack: SigLIP2 Top-50 → Qwen rerank
    if USE_QWEN:
        print(f"\n{'='*60}")
        print(f"[4/{n_models}] Reranking Top-{TOPK_FOR_RERANK} with {QWEN_MODEL_ID}")
        print(f"{'='*60}")
        t0 = time.time()
        sim_sig = cosine_sim(txt_sig, img_sig)
        sim_reranked = rerank_with_qwen(sim_sig, image_paths, categories)
        swe = sweep_thresholds(sim_reranked, y_true)
        swe["mrr"] = mrr(sim_reranked, y_true)
        results["Stack_SigLIP2_plus_Qwen"] = swe
        print(f"  Done in {time.time()-t0:.1f}s  |  F1={swe['max_f1']:.4f}  MRR={swe['mrr']:.4f}")

    # Model size info for the table
    model_sizes = {
        "CLIP_ViT_L_14": "Large (428M)",
        "CLIP_ViT_B_32_FT": "Base (151M)",
    }
    # Detect SigLIP size from model id
    if "SO400M" in siglip_id.upper() or "so400m" in siglip_id.lower():
        model_sizes[siglip_label] = "So400m (428M)"
    elif "L-16" in siglip_id or "l-16" in siglip_id.lower():
        model_sizes[siglip_label] = "Large (400M)"
    elif "B-16" in siglip_id or "b-16" in siglip_id.lower():
        model_sizes[siglip_label] = "Base (150M)"
    else:
        model_sizes[siglip_label] = siglip_id
    if "Stack_SigLIP2_plus_Qwen" in results:
        model_sizes["Stack_SigLIP2_plus_Qwen"] = model_sizes.get(siglip_label, "?") + " + 3B VLM"

    # Save artifacts
    print(f"\n{'='*60}")
    print("Saving artifacts...")
    print(f"{'='*60}")

    # Markdown table
    md_path = RESULTS_DIR / "comparison.md"
    header = "| Model | Model Size | Optimal Threshold | Max F1 | MRR |"
    sep    = "|---|---|---:|---:|---:|"
    rows: list[str] = []
    for key, val in results.items():
        sz = model_sizes.get(key, "?")
        rows.append(f"| {key} | {sz} | {val['optimal_threshold']:.2f} | {val['max_f1']:.4f} | {val['mrr']:.4f} |")

    with open(md_path, "w") as f:
        f.write(header + "\n")
        f.write(sep + "\n")
        for r in rows:
            f.write(r + "\n")
    print(f"  -> Markdown table saved to {md_path}")

    # Print summary to console
    print(f"\n{'='*60}")
    print("RESULTS SUMMARY")
    print(f"{'='*60}")
    print(header)
    print(sep)
    for r in rows:
        print(r)

    # JSON detailed report — ALL categories
    report = {"models": results}

    def per_query_errors(sim: np.ndarray, qname: str, threshold: float) -> dict[str, int]:
        qi = categories.index(qname)
        y = y_true[qi]
        pred = (sim[qi] >= threshold).astype(np.int32)
        fp = int(((pred == 1) & (y == 0)).sum())
        fn = int(((pred == 0) & (y == 1)).sum())
        return {"false_positives": fp, "false_negatives": fn}

    # Precompute all sims
    sims = {
        "CLIP_ViT_L_14": cosine_sim(txt_clip_l14, img_clip_l14),
        "CLIP_ViT_B_32_FT": cosine_sim(txt_ft, img_ft),
        siglip_label: cosine_sim(txt_sig, img_sig),
    }
    if USE_QWEN and "Stack_SigLIP2_plus_Qwen" in results:
        sims["Stack_SigLIP2_plus_Qwen"] = sim_reranked  # reuse from step 4

    report_specific: dict[str, dict[str, dict[str, int]]] = {}
    for model_key, vals in results.items():
        sim = sims[model_key]
        th = float(vals["optimal_threshold"])
        report_specific[model_key] = {
            cat: per_query_errors(sim, cat, th) for cat in categories
        }

    report["query_error_report"] = report_specific

    json_path = RESULTS_DIR / "evaluation_report.json"
    with open(json_path, "w") as f:
        json.dump(report, f, indent=2)
    print(f"  -> JSON report saved to {json_path}")

    elapsed = time.time() - total_start
    print(f"\nTotal evaluation time: {elapsed/60:.1f} min ({elapsed:.0f}s)")
    print(f"Dataset: {len(categories)} categories, {len(image_paths)} images")
    print("Done.")


if __name__ == "__main__":
    main()
