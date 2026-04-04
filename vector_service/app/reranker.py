import json
import os
from typing import List, Dict

class HybridRanker:
    def __init__(self, weights_path: str = "production_weights.json"):
        with open(weights_path, "r") as f:
            data = json.load(f)
            # We grab the averaged weights we optimized
            w = data["recommended_averaged_weights"]
            self.w1 = w["w1"] # Image Vector Weight
            self.w2 = w["w2"] # Text Vector Weight
            self.w3 = w["w3"] # Lexical Tag Weight

    def calculate_lexical_score(self, query_text: str, target_tags: List[str]) -> int:
        """Simple word overlap count for w3 tie-breaking."""
        query_words = set(query_text.lower().split())
        tag_set = set([t.lower() for t in target_tags])
        return len(query_words.intersection(tag_set))

    def get_hybrid_score(self, img_sim: float, txt_sim: float, lexical_count: int) -> float:
        """The final formula: w1*Img + w2*Txt + w3*Lex"""
        return (self.w1 * img_sim) + (self.w2 * txt_sim) + (self.w3 * lexical_count)

# Singleton instance
ranker = HybridRanker()

