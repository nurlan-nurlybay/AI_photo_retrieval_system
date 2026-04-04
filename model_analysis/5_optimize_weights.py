import numpy as np
import json
import optuna
from sklearn.model_selection import KFold
from sklearn.metrics.pairwise import cosine_similarity

img_vecs = np.load("siglip_image_vectors.npy", allow_pickle=True).item()
txt_vecs = np.load("siglip_text_vectors.npy", allow_pickle=True).item()
with open("metadata_prod_qwen.json", "r") as f:
    qwen_meta = json.load(f)
with open("metadata_gt_gemma.json", "r") as f:
    gemma_gt = json.load(f)

# Ensure intersection of all processed files
valid_files = [f for f in img_vecs.keys() if f in qwen_meta and f in gemma_gt]
X = np.array(valid_files)

def get_lexical_score(query_desc, target_meta):
    q_words = set(query_desc.lower().split())
    t_words = set(target_meta.get('description', '').lower().split() + target_meta.get('tags', []))
    return len(q_words.intersection(t_words))

def calculate_recall_at_1(weights, train_idx, val_idx):
    w1, w2, w3 = weights
    correct = 0
    
    for v_idx in val_idx:
        target_file = X[v_idx]
        query_text = gemma_gt[target_file].get('description', '') 
        q_vec = txt_vecs[target_file] 
        
        best_score = -1
        best_match = None
        
        for c_idx in np.concatenate((train_idx, val_idx)):
            cand_file = X[c_idx]
            sim_img = cosine_similarity(q_vec, img_vecs[cand_file])[0][0]
            sim_txt = cosine_similarity(q_vec, txt_vecs[cand_file])[0][0]
            lex_k = get_lexical_score(query_text, qwen_meta[cand_file])
            
            total_sim = (w1 * sim_img) + (w2 * sim_txt) + (w3 * lex_k)
            if total_sim > best_score:
                best_score = total_sim
                best_match = cand_file
                
        if best_match == target_file: correct += 1
            
    return correct / len(val_idx)

outer_cv = KFold(n_splits=5, shuffle=True, random_state=42)
inner_cv = KFold(n_splits=5, shuffle=True, random_state=42)
outer_results = []

for outer_fold, (train_val_idx, test_idx) in enumerate(outer_cv.split(X)):
    print(f"--- Outer Fold {outer_fold+1} ---")
    
    def objective(trial):
        w1 = trial.suggest_float("w1", 0.0, 1.0)
        w2 = trial.suggest_float("w2", 0.0, 1.0)
        w3 = trial.suggest_float("w3", 0.0, 0.5) 
        
        inner_recalls = []
        for inner_train_idx, inner_val_idx in inner_cv.split(train_val_idx):
            abs_train = train_val_idx[inner_train_idx]
            abs_val = train_val_idx[inner_val_idx]
            inner_recalls.append(calculate_recall_at_1((w1, w2, w3), abs_train, abs_val))
        return np.mean(inner_recalls)

    study = optuna.create_study(direction="maximize")
    study.optimize(objective, n_trials=30)
    
    best_weights = study.best_params
    print(f"Best inner weights: {best_weights}")
    outer_results.append({
        "fold": outer_fold + 1,
        "weights": best_weights,
        "test_idx": test_idx.tolist(),
        "train_val_idx": train_val_idx.tolist()
    })

with open("tuned_weights.json", "w") as f:
    json.dump(outer_results, f, indent=4)

