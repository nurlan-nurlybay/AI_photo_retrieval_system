import json
class HybridRanker:
    def __init__(self, weights_path: str = "production_weights.json"):
        with open(weights_path, "r") as f:
            data = json.load(f)
            w = data["recommended_averaged_weights"]
            self.w1 = w["w1"] 
            self.w2 = w["w2"] 
            self.w3 = w["w3"] 

    def calculate_lexical_score(self, query_text: str, target_tags: list[str]) -> int:
        query_words = set(query_text.lower().split())
        tag_set = set([t.lower() for t in target_tags])
        return len(query_words.intersection(tag_set))

    def get_hybrid_score(self, img_sim: float, txt_sim: float, lexical_count: int) -> float:
        return (self.w1 * img_sim) + (self.w2 * txt_sim) + (self.w3 * lexical_count)

    def rerank(self, query_text: str, img_hits: list[dict], txt_hits: list[dict], top_k: int) -> list[dict]:
        # Create a fast lookup dict for text hits by ID
        txt_lookup = {hit["id"]: hit for hit in txt_hits}
        
        final_scores = []
        for img in img_hits:
            img_id = img["id"]
            img_score = img["score"]
            
            # If the image has a matching text vector, fuse the scores
            if img_id in txt_lookup:
                txt_data = txt_lookup[img_id]
                lex_score = self.calculate_lexical_score(query_text, txt_data.get("tags", []))
                final_score = self.get_hybrid_score(img_score, txt_data["score"], lex_score)
            else:
                # Fallback to naive score if text data is missing for this specific image
                final_score = img_score
                
            final_scores.append({"id": img_id, "score": final_score})
            
        # Sort highest score first
        final_scores.sort(key=lambda x: x["score"], reverse=True)
        return final_scores[:top_k]

# Singleton instance
ranker = HybridRanker()
