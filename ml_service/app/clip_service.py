import threading

import torch
import numpy as np
from transformers import CLIPProcessor, CLIPModel
from PIL import Image
from io import BytesIO
from peft.peft_model import PeftModel
from .models import ModelName, EncodeOptions

device = "cuda" if torch.cuda.is_available() else "cpu"

# Cache models & processors by name so we only download/init once
_MODELS: dict[str, CLIPModel] = {}
_PROCS: dict[str, CLIPProcessor] = {}
_LOCK = threading.Lock()
TUNED_MODEL_PATH = "fine_tuning/clip_finetuned_epoch_3"


def warmup() -> None:
    """Eagerly load the default model so concurrent requests don't race."""
    _get_model_and_processor(ModelName.vit_b32, quantize=True)


def _get_model_and_processor(name: ModelName, quantize: bool) -> tuple[CLIPModel, CLIPProcessor]:
    key = f"{name.value}_{'tuned' if TUNED_MODEL_PATH else 'base'}_{'quantized' if quantize else 'full'}"  # Cache separately
    m = _MODELS.get(key)
    p = _PROCS.get(key)
    if m is not None and p is not None:
        return m, p  # type: ignore

    with _LOCK:
        # Double-check after acquiring lock
        m = _MODELS.get(key)
        p = _PROCS.get(key)
        if m is not None and p is not None:
            return m, p  # type: ignore

        m = CLIPModel.from_pretrained(name.value).to(device) # type: ignore
        p = CLIPProcessor.from_pretrained(name.value)

        if TUNED_MODEL_PATH:
            m = PeftModel.from_pretrained(m, TUNED_MODEL_PATH)
            m = m.merge_and_unload()
            m = m.to(device) # type: ignore

        if quantize:
            m = torch.quantization.quantize_dynamic(
                m, {torch.nn.Linear}, dtype=torch.qint8
            )
        _MODELS[key] = m # type: ignore
        _PROCS[key] = p # type: ignore
    return m, p # type: ignore


def _l2_normalize(arr: np.ndarray) -> np.ndarray:
    arr = arr.astype(np.float32, copy=False)
    n = np.linalg.norm(arr, axis=-1, keepdims=True)
    n[n == 0] = 1.0
    return arr / n


# -------- Text (batch) --------
def encode_text(texts: list[str], opts: EncodeOptions) -> list[list[float]]:
    """
    Batch text → embeddings. For a single query, pass ["query"].
    Returns: list of 512-float vectors, one per input text.
    """
    model, processor = _get_model_and_processor(opts.model, opts.quantize)
    inputs = processor(text=texts, return_tensors="pt", padding=True, truncation=True).to(device)
    with torch.inference_mode():
        vecs = model.get_text_features(**inputs)  # (B, 512) # type: ignore
    arr = vecs.detach().cpu().numpy()
    if opts.normalize:
        arr = _l2_normalize(arr)
    return arr.tolist()


# -------- Image (batch) --------
def encode_image(images: list[bytes], opts: EncodeOptions) -> list[list[float]]:
    """
    Batch images → embeddings. For a single image, pass [image_bytes].
    Returns: list of 512-float vectors, one per image.
    """
    model, processor = _get_model_and_processor(opts.model, opts.quantize)
    pil_list = [Image.open(BytesIO(b)).convert("RGB") for b in images]
    inputs = processor(images=pil_list, return_tensors="pt", padding=True).to(device)
    with torch.inference_mode():
        vecs = model.get_image_features(**inputs)  # (B, 512) # type: ignore
    arr = vecs.detach().cpu().numpy()
    if opts.normalize:
        arr = _l2_normalize(arr)
    return arr.tolist()
