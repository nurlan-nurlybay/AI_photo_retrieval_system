from pathlib import Path
import numpy as np
from PIL import Image


def list_images(root: Path) -> tuple[list[Path], list[str]]:
    image_paths: list[Path] = []
    labels: list[str] = []
    for category_dir in sorted([p for p in root.iterdir() if p.is_dir()]):
        for img in sorted(category_dir.glob("*.png")) + sorted(category_dir.glob("*.jpg")):
            image_paths.append(img)
            labels.append(category_dir.name)
    return image_paths, labels


def load_image(path: Path) -> Image.Image:
    return Image.open(path).convert("RGB")


def normalize_rows(x: np.ndarray, eps: float = 1e-12) -> np.ndarray:
    norms = np.linalg.norm(x, axis=1, keepdims=True)
    norms = np.maximum(norms, eps)
    return x / norms


def cosine_sim(a: np.ndarray, b: np.ndarray) -> np.ndarray:
    # a: [N, D], b: [M, D]
    a_n = normalize_rows(a)
    b_n = normalize_rows(b)
    return a_n @ b_n.T


def compute_f1_at_threshold(sim: np.ndarray, y_true: np.ndarray, threshold: float) -> float:
    # sim: [num_queries, num_images], y_true: [num_queries, num_images]
    y_pred = (sim >= threshold).astype(np.int32)
    tp = (y_pred * y_true).sum()
    fp = (y_pred * (1 - y_true)).sum()
    fn = ((1 - y_pred) * y_true).sum()
    precision = tp / (tp + fp + 1e-8)
    recall = tp / (tp + fn + 1e-8)
    if precision + recall == 0:
        return 0.0
    return 2 * precision * recall / (precision + recall)


def mrr(sim: np.ndarray, y_true: np.ndarray) -> float:
    # For each query, rank images by sim and find the first relevant
    num_q = sim.shape[0]
    rr = []
    for i in range(num_q):
        scores = sim[i]
        rel = y_true[i]
        order = np.argsort(-scores)
        ranks = np.where(rel[order] == 1)[0]
        if len(ranks) == 0:
            rr.append(0.0)
        else:
            rr.append(1.0 / (ranks[0] + 1))
    return float(np.mean(rr))
