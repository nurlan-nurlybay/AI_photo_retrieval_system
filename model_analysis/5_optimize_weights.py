import numpy as np
import json
import optuna
from sklearn.model_selection import KFold

# 1. Load Data
print("Loading vectors and metadata...")
img_vecs = np.load("siglip_image_vectors.npy", allow_pickle=True).item()
txt_vecs = np.load("siglip_text_vectors.npy", allow_pickle=True).item()
with open("metadata_prod_qwen.json", "r") as f:
    qwen_meta = json.load(f)
with open("metadata_gt_gemma.json", "r") as f:
    gemma_gt = json.load(f)

# Ensure intersection and stable sorting for reproducibility
valid_files = sorted([f for f in img_vecs.keys() if f in qwen_meta and f in gemma_gt])
X = np.array(valid_files)
N = len(X)
print(f"Loaded {N} valid files. Precomputing matrices...")

# 2. Build Matrices
I_mat = np.zeros((N, 1152))
T_mat = np.zeros((N, 1152))
for i, f in enumerate(valid_files):
    # Handle cases where vectors might be wrapped in extra dims
    I_mat[i] = img_vecs[f][0] if np.ndim(img_vecs[f]) > 1 else img_vecs[f]
    T_mat[i] = txt_vecs[f][0] if np.ndim(txt_vecs[f]) > 1 else txt_vecs[f]

# 3. Precompute Similarity Matrices
# (Query Text @ Image Target) and (Query Text @ Text Target)
S_img = T_mat @ I_mat.T  
S_txt = T_mat @ T_mat.T  

# 4. Precompute Lexical Matrix (Word overlap)
S_lex = np.zeros((N, N))
# Gemma Ground Truth acts as the "User Query"
q_words_list = [set(gemma_gt[f].get('description', '').lower().split()) for f in valid_files]

# FIXED: Strictly bound to the 10 tags provided by Qwen (matching Vector Service behavior)
t_words_list = [set([tag.lower() for tag in qwen_meta[f].get('tags', [])]) for f in valid_files]

for i in range(N):
    qw = q_words_list[i]
    for j in range(N):
        S_lex[i, j] = len(qw.intersection(t_words_list[j]))

print("Matrices precomputed. Starting Nested K-Fold Optimization...")

# 5. Nested CV Setup
outer_cv = KFold(n_splits=5, shuffle=True, random_state=42)
inner_cv = KFold(n_splits=5, shuffle=True, random_state=42)

all_fold_details = []
all_oof_predictions = {}
optuna.logging.set_verbosity(optuna.logging.WARNING)

for outer_fold, (train_val_idx, test_idx) in enumerate(outer_cv.split(X)):
    print(f"\n--- Processing Outer Fold {outer_fold+1}/5 ---")
    
    def objective(trial):
        w1 = trial.suggest_float("w1", 0.0, 1.0)
        w2 = trial.suggest_float("w2", 0.0, 1.0)
        w3 = trial.suggest_float("w3", 0.0, 0.5) 
        
        inner_recalls = []
        for inner_train_idx, inner_val_idx in inner_cv.split(train_val_idx):
            abs_val = train_val_idx[inner_val_idx]
            # Search pool for this inner fold is the train_val set
            sim = (w1 * S_img[abs_val, :][:, train_val_idx]) + \
                  (w2 * S_txt[abs_val, :][:, train_val_idx]) + \
                  (w3 * S_lex[abs_val, :][:, train_val_idx])
            
            # Count correct hits (where argmax in the pool matches the target's relative index)
            best_match_rel = np.argmax(sim, axis=1)
            best_match_abs = train_val_idx[best_match_rel]
            correct = np.sum(best_match_abs == abs_val)
            inner_recalls.append(correct / len(abs_val))
        return np.mean(inner_recalls)

    # A. Tune on Inner Folds
    study = optuna.create_study(direction="maximize")
    study.optimize(objective, n_trials=30)
    best_params = study.best_params
    
    # B. Apply Best Weights to the Unseen Test Fold (OOF)
    w1, w2, w3 = best_params["w1"], best_params["w2"], best_params["w3"]
    # We search the ENTIRE dataset for the test queries to be realistic
    test_sim = (w1 * S_img[test_idx, :]) + (w2 * S_txt[test_idx, :]) + (w3 * S_lex[test_idx, :])
    top_10_indices = np.argsort(-test_sim, axis=1)[:, :10]
    
    correct_outer = 0
    for i, t_id in enumerate(test_idx):
        query_file = X[t_id]
        top_10_files = X[top_10_indices[i]].tolist()
        all_oof_predictions[query_file] = top_10_files
        
        # Calculate strict Recall@1 for this specific outer test fold
        if query_file == top_10_files[0]:
            correct_outer += 1
            
    outer_score = correct_outer / len(test_idx)

    print(f"Fold {outer_fold+1} Best Weights: {best_params} (Outer OOF Score: {outer_score:.4f})")
    
    all_fold_details.append({
        "fold": outer_fold + 1,
        "weights": best_params,
        "outer_test_recall": float(outer_score)
    })

# 6. Final Aggregation
avg_weights = {
    "w1": float(np.mean([f["weights"]["w1"] for f in all_fold_details])),
    "w2": float(np.mean([f["weights"]["w2"] for f in all_fold_details])),
    "w3": float(np.mean([f["weights"]["w3"] for f in all_fold_details]))
}

# 7. Save Everything Clearly
# This file is for the Machine (and your review of the folds)
final_production_package = {
    "recommended_averaged_weights": avg_weights,
    "per_fold_details": all_fold_details
}

with open("production_weights.json", "w") as f:
    json.dump(final_production_package, f, indent=4)

# This file is ONLY for the evaluation script
with open("evaluation_data.json", "w") as f:
    json.dump(all_oof_predictions, f, indent=4)

print("\n" + "="*50)
print("[SUCCESS] Optimization Complete.")
print(f"Final Averaged Weights: {avg_weights}")
print("Check 'production_weights.json' for the full per-fold breakdown.")
print("="*50)

