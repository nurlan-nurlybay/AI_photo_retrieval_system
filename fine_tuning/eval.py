import os
import torch
import sys
from torch.utils.data import Dataset, DataLoader
from PIL import Image
from transformers import CLIPProcessor, CLIPModel
from peft.peft_model import PeftModel
from tqdm import tqdm
from typing import cast

# --- CONFIGURATION ---
TEST_IMAGE_DIR = "data/USED/test_images"
TEST_METADATA_DIR = "data/USED/CSV files for SED-EImm/EiMM_txt/EiMM_test_txt"

MODEL_NAME = "openai/clip-vit-base-patch32"
BATCH_SIZE = 64
device = "cuda" if torch.cuda.is_available() else "cpu"

class TestDataset(Dataset):
    def __init__(self, image_root, metadata_dir, processor):
        self.image_root = image_root
        self.processor = processor
        self.samples = []
        self.class_names = []

        # 1. Detect Classes from filenames (e.g. "concert_test.txt" -> "concert")
        try:
            txt_files = [f for f in os.listdir(metadata_dir) if f.endswith("_test.txt")]
        except FileNotFoundError:
            print(f"❌ Metadata dir not found: {metadata_dir}")
            return

        print(f"📂 Found {len(txt_files)} test categories.")

        for txt_file in txt_files:
            label = txt_file.replace("_test.txt", "")
            self.class_names.append(label)

            path = os.path.join(metadata_dir, txt_file)
            with open(path, 'r') as f:
                for line in f:
                    clean_line = line.strip()
                    if not clean_line: continue

                    if "," in clean_line:
                        filename = clean_line.split(",")[0].strip()
                    else:
                        filename = clean_line.split(" ")[0].strip()

                    full_path = os.path.join(self.image_root, filename)
                    # Only add if file exists to avoid crashing later
                    if os.path.exists(full_path):
                        label_idx = self.class_names.index(label)
                        self.samples.append((full_path, label_idx))

        print(f"✅ Loaded {len(self.samples)} valid test images.")
        print(f"📋 Categories: {self.class_names}")

    def __len__(self):
        return len(self.samples)

    def __getitem__(self, idx):
        img_path, label_idx = self.samples[idx]
        try:
            image = Image.open(img_path).convert("RGB")
            # Using the processor callable
            processed = self.processor(images=image, return_tensors="pt")
            return {
                "pixel_values": processed["pixel_values"][0],
                "label_idx": label_idx
            }
        except Exception:
            return self.__getitem__((idx + 1) % len(self))

def evaluate(model_path=None):
    print(f"🧠 Loading Base Model: {MODEL_NAME}")

    base_model = CLIPModel.from_pretrained(MODEL_NAME)
    model = base_model.to(device) # type: ignore

    processor_raw = CLIPProcessor.from_pretrained(MODEL_NAME)
    if isinstance(processor_raw, tuple):
        processor_raw = processor_raw[0]
    processor = cast(CLIPProcessor, processor_raw)

    # Load Fine-Tuned Adapters if provided
    if model_path:
        print(f"🔋 Loading LoRA Adapter from: {model_path}")
        model = PeftModel.from_pretrained(model, model_path)
        model = model.merge_and_unload()
        model = model.to(device) # type: ignore

    model.eval() # type: ignore

    # Prepare Data
    dataset = TestDataset(TEST_IMAGE_DIR, TEST_METADATA_DIR, processor)

    if len(dataset) == 0:
        print("❌ Error: No test images found. Check your paths.")
        return

    dataloader = DataLoader(dataset, batch_size=BATCH_SIZE, num_workers=0, shuffle=False)

    print("🔤 Computing Text Embeddings...")
    class_descriptions = [f"A photo of a {c.replace('_', ' ')}" for c in dataset.class_names]

    # Fix 2 Cont: Processor is now explicitly cast as CLIPProcessor
    text_inputs = processor(text=class_descriptions, return_tensors="pt", padding=True)
    text_inputs = text_inputs.to(device) # type: ignore

    with torch.inference_mode():
        text_features = model.get_text_features(**text_inputs) # type: ignore
        text_features /= text_features.norm(dim=-1, keepdim=True)

    print("🚀 Running Evaluation...")
    correct = 0
    total = 0

    for batch in tqdm(dataloader):
        pixel_values = batch["pixel_values"].to(device)
        labels = batch["label_idx"].to(device)

        with torch.inference_mode():
            image_features = model.get_image_features(pixel_values) # type: ignore
            image_features /= image_features.norm(dim=-1, keepdim=True)

            similarity = (100.0 * image_features @ text_features.T).softmax(dim=-1)
            predictions = similarity.argmax(dim=-1)

            correct += (predictions == labels).sum().item()
            total += labels.size(0)

    if total > 0:
        accuracy = 100 * correct / total
        print("\n📊 RESULTS:")
        print(f"   Model: {'Baseline (Zero-Shot)' if not model_path else model_path}")
        print(f"   Accuracy: {accuracy:.2f}%")
        print(f"   Correct: {correct}/{total}")
    else:
        print("❌ Total evaluated images is 0.")

if __name__ == "__main__":
    adapter_path = sys.argv[1] if len(sys.argv) > 1 else None
    evaluate(adapter_path)
