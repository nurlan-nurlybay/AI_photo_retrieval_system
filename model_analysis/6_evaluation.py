import json
import numpy as np

print("Loading data...")
with open("evaluation_data.json", "r") as f:
    oof_preds = json.load(f)

# The keys in our eval data represent the entire 1468 dataset across all folds
valid_files = sorted(list(oof_preds.keys()))
N = len(valid_files)

img_vecs = np.load("siglip_image_vectors.npy", allow_pickle=True).item()
txt_vecs = np.load("siglip_text_vectors.npy", allow_pickle=True).item()

print("Computing SigLIP baseline matrices...")
I_mat = np.zeros((N, 1152))
T_mat = np.zeros((N, 1152))

for i, f in enumerate(valid_files):
    I_mat[i] = img_vecs[f][0] if np.ndim(img_vecs[f]) > 1 else img_vecs[f]
    T_mat[i] = txt_vecs[f][0] if np.ndim(txt_vecs[f]) > 1 else txt_vecs[f]

# Baseline: Query Text Vector dot Target Image Vector
S_img = T_mat @ I_mat.T

base_1, base_5, base_10 = 0, 0, 0
hyb_1, hyb_5, hyb_10 = 0, 0, 0

print("Tallying comparative results...\n")
for i, target_file in enumerate(valid_files):
    # --- BASELINE TALLY (Image Sim Only) ---
    base_scores = S_img[i, :]
    base_top_10_idx = np.argsort(-base_scores)[:10]
    base_top_10_files = [valid_files[idx] for idx in base_top_10_idx]
    
    if target_file in base_top_10_files[:1]: base_1 += 1
    if target_file in base_top_10_files[:5]: base_5 += 1
    if target_file in base_top_10_files[:10]: base_10 += 1

    # --- HYBRID TALLY (From your JSON) ---
    hyb_top_10 = oof_preds[target_file]
    if target_file in hyb_top_10[:1]: hyb_1 += 1
    if target_file in hyb_top_10[:5]: hyb_5 += 1
    if target_file in hyb_top_10[:10]: hyb_10 += 1

# Prepare the data structure for saving
results = {
    "summary": {
        "total_test_queries": N,
        "metrics": ["Recall@1", "Recall@5", "Recall@10"]
    },
    "baseline_siglip": {
        "recall_1": round(base_1/N, 4),
        "recall_5": round(base_5/N, 4),
        "recall_10": round(base_10/N, 4)
    },
    "hybrid_oof": {
        "recall_1": round(hyb_1/N, 4),
        "recall_5": round(hyb_5/N, 4),
        "recall_10": round(hyb_10/N, 4)
    },
    "improvement_delta": {
        "recall_1": round((hyb_1 - base_1)/N, 4),
        "recall_5": round((hyb_5 - base_5)/N, 4),
        "recall_10": round((hyb_10 - base_10)/N, 4)
    }
}

# 1. SAVE TO FILE
with open("final_evaluation_metrics.json", "w") as f:
    json.dump(results, f, indent=4)

# 2. PRINT TO TERMINAL (For confirmation)
print("="*45)
print(f"{'Metric':<15} | {'SigLIP Baseline':<12} | {'Hybrid (OOF)':<12}")
print("-" * 45)
print(f"{'Recall@1':<15} | {results['baseline_siglip']['recall_1']*100:>11.2f}% | {results['hybrid_oof']['recall_1']*100:>11.2f}%")
print(f"{'Recall@5':<15} | {results['baseline_siglip']['recall_5']*100:>11.2f}% | {results['hybrid_oof']['recall_5']*100:>11.2f}%")
print(f"{'Recall@10':<15} | {results['baseline_siglip']['recall_10']*100:>11.2f}% | {results['hybrid_oof']['recall_10']*100:>11.2f}%")
print("="*45)
print(f"[SUCCESS] Metrics saved to final_evaluation_metrics.json")

