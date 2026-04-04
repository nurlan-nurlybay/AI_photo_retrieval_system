import numpy as np
import json
from sklearn.metrics.pairwise import cosine_similarity

img_vecs = np.load("siglip_image_vectors.npy", allow_pickle=True).item()
txt_vecs = np.load("siglip_text_vectors.npy", allow_pickle=True).item()
with open("metadata_prod_qwen.json", "r") as f:
    qwen_meta = json.load(f)
with open("metadata_gt_gemma.json", "r") as f:
    gemma_gt = json.load(f)
with open("tuned_weights.json", "r") as f:
    outer_folds = json.load(f)

valid_files = [f for f in img_vecs.keys() if f in qwen_meta and f in gemma_gt]
X = np.array(valid_files)

def get_lexical_score(query_desc, target_meta):
    q_words = set(query_desc.lower().split())
    t_words = set(target_meta.get('description', '').lower().split() + target_meta.get('tags', []))
    return len(q_words.intersection(t_words))

total_test_queries = 0
total_correct_at_1 = 0
total_correct_at_5 = 0
total_correct_at_10 = 0

for fold_data in outer_folds:
    w1 = fold_data['weights']['w1']
    w2 = fold_data['weights']['w2']
    w3 = fold_data['weights']['w3']
    
    test_idx = fold_data['test_idx']
    train_val_idx = fold_data['train_val_idx']
    search_pool = np.concatenate((train_val_idx, test_idx))
    
    for t_idx in test_idx:
        target_file = X[t_idx]
        query_text = gemma_gt[target_file].get('description', '')
        q_vec = txt_vecs[target_file] 
        
        scores = []
        for c_idx in search_pool:
            cand_file = X[c_idx]
            sim_img = cosine_similarity(q_vec, img_vecs[cand_file])[0][0]
            sim_txt = cosine_similarity(q_vec, txt_vecs[cand_file])[0][0]
            lex_k = get_lexical_score(query_text, qwen_meta[cand_file])
            
            total_sim = (w1 * sim_img) + (w2 * sim_txt) + (w3 * lex_k)
            scores.append((total_sim, cand_file))
            
        scores.sort(key=lambda x: x[0], reverse=True)
        top_10 = [s[1] for s in scores[:10]]
        
        if target_file in top_10[:1]: total_correct_at_1 += 1
        if target_file in top_10[:5]: total_correct_at_5 += 1
        if target_file in top_10[:10]: total_correct_at_10 += 1
        total_test_queries += 1

print(f"Final OOF Hybrid Recall@1:  {total_correct_at_1 / total_test_queries * 100:.2f}%")
print(f"Final OOF Hybrid Recall@5:  {total_correct_at_5 / total_test_queries * 100:.2f}%")
print(f"Final OOF Hybrid Recall@10: {total_correct_at_10 / total_test_queries * 100:.2f}%")

