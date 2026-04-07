from __future__ import annotations
import os
from PIL import Image
import torch

# Wrapper around Qwen2-VL to assign a scalar relevance score in [0,1]
# Uses logits-based scoring (Yes/No softmax) instead of text generation
# to avoid "Yes-Man" bias in small VLMs.
# Requires: transformers>=4.45, accelerate, bitsandbytes (optional for 4-bit), pillow, qwen-vl-utils

DEFAULT_MODEL_ID = os.environ.get("QWEN_VL_MODEL", "Qwen/Qwen2-VL-2B-Instruct")
QUANT_4BIT = os.environ.get("QWEN_QUANT_4BIT", "1") == "1"


class QwenReranker:
    def __init__(self, model_id: str = DEFAULT_MODEL_ID):
        from transformers import AutoProcessor, BitsAndBytesConfig, Qwen2VLForConditionalGeneration

        load_kwargs: dict = {"trust_remote_code": True}
        if QUANT_4BIT:
            load_kwargs["quantization_config"] = BitsAndBytesConfig(
                load_in_4bit=True,
                bnb_4bit_compute_dtype=torch.float16,
            )
            load_kwargs["device_map"] = "auto"
        else:
            load_kwargs["torch_dtype"] = torch.float16
            load_kwargs["device_map"] = "auto"

        self.model = Qwen2VLForConditionalGeneration.from_pretrained(model_id, **load_kwargs)
        self.processor = AutoProcessor.from_pretrained(model_id, trust_remote_code=True)
        self.model.eval()

        # Pre-compute token IDs for "Yes" and "No"
        self.yes_token_id = self.processor.tokenizer.encode("Yes", add_special_tokens=False)[0]
        self.no_token_id = self.processor.tokenizer.encode("No", add_special_tokens=False)[0]

    @torch.no_grad()
    def score(self, image: Image.Image, query: str) -> float:
        messages = [
            {
                "role": "user",
                "content": [
                    {"type": "image", "image": image},
                    {"type": "text", "text": (
                        f"Question: Does this image clearly show '{query}'? "
                        "Answer with only Yes or No.\nAnswer:"
                    )},
                ],
            }
        ]

        text_prompt = self.processor.apply_chat_template(messages, tokenize=False, add_generation_prompt=True)
        from qwen_vl_utils import process_vision_info
        vision_info = process_vision_info(messages)
        if len(vision_info) == 3:
            image_inputs, video_inputs, _ = vision_info
        else:
            image_inputs, video_inputs = vision_info

        inputs = self.processor(
            text=[text_prompt],
            images=image_inputs,
            videos=video_inputs,
            padding=True,
            return_tensors="pt",
        ).to(self.model.device)

        # Forward pass only (no generation) to get raw logits
        outputs = self.model(**inputs)

        # Get logits of the very last token (where the answer should be)
        last_token_logits = outputs.logits[0, -1, :]

        yes_score = last_token_logits[self.yes_token_id].item()
        no_score = last_token_logits[self.no_token_id].item()

        # Softmax between Yes/No only → probability of "Yes"
        probs = torch.softmax(torch.tensor([yes_score, no_score]), dim=0)
        return probs[0].item()

    def batch_score(self, images: list[Image.Image], query: str) -> list[float]:
        from tqdm import tqdm
        return [self.score(img, query) for img in tqdm(images, desc=f"  Scoring '{query}'", leave=False, unit="img")]
