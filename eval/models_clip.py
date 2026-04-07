from pathlib import Path

import torch
import numpy as np
from PIL import Image


def _to_device(x, device):
    if isinstance(x, (list, tuple)):
        return [xi.to(device) if hasattr(xi, 'to') else xi for xi in x]
    return x.to(device) if hasattr(x, 'to') else x


class OpenCLIPEncoder:
    def __init__(self, model_name: str, pretrained: str | None = None):
        import open_clip
        self.device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
        if pretrained is None:
            self.model, self.preprocess_train, self.preprocess = open_clip.create_model_and_transforms(model_name)
        else:
            self.model, self.preprocess_train, self.preprocess = open_clip.create_model_and_transforms(model_name, pretrained=pretrained)
        self.tokenizer = open_clip.get_tokenizer(model_name)
        # Monkey-patch HF tokenizer to have batch_encode_plus if missing (Transformers 5.x)
        try:
            if hasattr(self.tokenizer, 'tokenizer'):
                hf_tok = getattr(self.tokenizer, 'tokenizer')
                if not hasattr(hf_tok, 'batch_encode_plus') and hasattr(hf_tok, '__call__'):
                    setattr(hf_tok, 'batch_encode_plus', hf_tok.__call__)
        except Exception:
            pass
        self.model = self.model.to(self.device)
        self.model.eval()

    @torch.no_grad()
    def encode_images(self, images: list[Image.Image], batch_size: int = 32) -> np.ndarray:
        feats = []
        for i in range(0, len(images), batch_size):
            batch_imgs = images[i:i+batch_size]
            batch_tensor = torch.stack([self.preprocess(im) for im in batch_imgs]).to(self.device)
            with torch.amp.autocast(device_type="cuda", enabled=torch.cuda.is_available()):
                emb = self.model.encode_image(batch_tensor)
                emb = emb / emb.norm(dim=-1, keepdim=True)
            feats.append(emb.detach().cpu())
        return torch.cat(feats, dim=0).numpy()

    @torch.no_grad()
    def encode_texts(self, texts: list[str], batch_size: int = 64) -> np.ndarray:
        feats = []
        for i in range(0, len(texts), batch_size):
            batch_txts = texts[i:i+batch_size]
            tokens = self.tokenizer(batch_txts)
            # Move to device
            if isinstance(tokens, dict):
                tokens = {k: (v.to(self.device) if hasattr(v, 'to') else v) for k, v in tokens.items()}
            else:
                tokens = tokens.to(self.device)
            with torch.amp.autocast(device_type="cuda", enabled=torch.cuda.is_available()):
                try:
                    emb = self.model.encode_text(tokens)
                except TypeError:
                    emb = self.model.encode_text(**tokens)  # type: ignore
                emb = emb / emb.norm(dim=-1, keepdim=True)
            feats.append(emb.detach().cpu())
        return torch.cat(feats, dim=0).numpy()


class HFCLIPWithPEFT:
    def __init__(self, base_model_name: str = "openai/clip-vit-base-patch32", adapter_dir: str | Path | None = None):
        from transformers import CLIPModel, CLIPProcessor
        from peft import PeftModel
        self.device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
        base = CLIPModel.from_pretrained(base_model_name)
        if adapter_dir is not None:
            peft_model = PeftModel.from_pretrained(base, str(adapter_dir))
            try:
                # Merge LoRA weights for clean inference with projected features
                self.model = peft_model.merge_and_unload()
            except Exception:
                self.model = peft_model
        else:
            self.model = base
        self.model.eval().to(self.device)
        self.processor = CLIPProcessor.from_pretrained(base_model_name)

    def _extract_feature(self, out, kind: str) -> torch.Tensor:
        # kind in {"image_embeds", "text_embeds"}
        if isinstance(out, torch.Tensor):
            return out
        # ModelOutput or dict-like
        if hasattr(out, kind):
            val = getattr(out, kind)
            if isinstance(val, torch.Tensor):
                return val
        if isinstance(out, dict) and kind in out:
            val = out[kind]
            if isinstance(val, torch.Tensor):
                return val
        # Fallbacks
        if hasattr(out, 'last_hidden_state') and isinstance(out.last_hidden_state, torch.Tensor):
            # CLS token or mean pool
            hidden = out.last_hidden_state
            if hidden.ndim == 3:
                return hidden.mean(dim=1)
        raise RuntimeError(f'Unexpected output type for {kind}')

    @torch.no_grad()
    def encode_images(self, images: list[Image.Image], batch_size: int = 32) -> np.ndarray:
        feats = []
        for i in range(0, len(images), batch_size):
            batch_imgs = images[i:i+batch_size]
            inputs = self.processor(images=batch_imgs, return_tensors="pt").to(self.device)
            with torch.amp.autocast(device_type="cuda", enabled=torch.cuda.is_available()):
                # Explicit path: vision backbone -> pooling -> projection
                vision_out = self.model.vision_model(pixel_values=inputs["pixel_values"])  # type: ignore
                if hasattr(vision_out, 'pooler_output') and vision_out.pooler_output is not None:
                    pooled = vision_out.pooler_output
                else:
                    pooled = vision_out.last_hidden_state.mean(dim=1)
                img = self.model.visual_projection(pooled)
                img = img / img.norm(dim=-1, keepdim=True)
            feats.append(img.detach().cpu())
        return torch.cat(feats, dim=0).numpy()

    @torch.no_grad()
    def encode_texts(self, texts: list[str], batch_size: int = 64) -> np.ndarray:
        feats = []
        for i in range(0, len(texts), batch_size):
            batch_txts = texts[i:i+batch_size]
            inputs = self.processor(text=batch_txts, return_tensors="pt", padding=True, truncation=True).to(self.device)
            with torch.amp.autocast(device_type="cuda", enabled=torch.cuda.is_available()):
                # Explicit path: text backbone -> pooling -> projection
                text_out = self.model.text_model(input_ids=inputs["input_ids"], attention_mask=inputs.get("attention_mask", None))  # type: ignore
                pooled = text_out.pooler_output
                txt = self.model.text_projection(pooled)
                txt = txt / txt.norm(dim=-1, keepdim=True)
            feats.append(txt.detach().cpu())
        return torch.cat(feats, dim=0).numpy()
