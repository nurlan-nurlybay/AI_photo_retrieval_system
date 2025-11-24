import os
import torch
from torch.utils.data import Dataset, DataLoader
from PIL import Image
from transformers import CLIPProcessor, CLIPModel
from peft.tuners.lora import LoraConfig
from peft.mapping import get_peft_model
from tqdm import tqdm
from torch.optim.adamw import AdamW
from typing import cast

# --- CONFIGURATION ---
# Path where you merged all training images (e.g., train-part1 + train-part2)
IMAGE_ROOT = "data/USED/train_images"

# Path to the metadata folder containing .txt files
METADATA_DIR = "data/USED/CSV files for SED-EImm/EiMM_txt/Eimm_train_txt"

MODEL_NAME = "openai/clip-vit-base-patch32"
BATCH_SIZE = 32
EPOCHS = 3
LEARNING_RATE = 1e-4
device = "cuda" if torch.cuda.is_available() else "cpu"

class FlatPhotoDataset(Dataset):
    def __init__(self, image_root, metadata_dir, processor):
        self.image_root = image_root
        self.processor = processor
        self.samples = []

        # 1. Scan metadata files
        if not os.path.exists(metadata_dir):
            raise ValueError(f"Metadata path not found: {metadata_dir}")

        txt_files = [f for f in os.listdir(metadata_dir) if f.endswith("_train.txt")]
        print(f"📂 Found {len(txt_files)} metadata files. Parsing...")

        for txt_file in txt_files:
            # Extract label from filename (e.g., 'concert_train.txt' -> 'concert')
            label = txt_file.replace("_train.txt", "")
            description = f"A photo of a {label.replace('_', ' ')}"

            path = os.path.join(metadata_dir, txt_file)
            with open(path, 'r') as f:
                lines = f.readlines()

            for line in lines:
                clean_line = line.strip()
                if not clean_line: continue

                # Split filename from label
                if "," in clean_line:
                    filename = clean_line.split(",")[0].strip()
                else:
                    filename = clean_line.split(" ")[0].strip()

                # Construct full path
                full_path = os.path.join(self.image_root, filename)
                self.samples.append((full_path, description))

        print(f"✅ Loaded {len(self.samples)} valid image-text pairs.")

    def __len__(self):
        return len(self.samples)

    def __getitem__(self, idx):
        img_path, text = self.samples[idx]
        try:
            image = Image.open(img_path).convert("RGB")

            # Using the processor (now strictly typed in main)
            processed = self.processor(
                text=[text],
                images=image,
                return_tensors="pt",
                padding="max_length",
                truncation=True,
                max_length=32
            )
            return {
                "input_ids": processed["input_ids"][0],
                "pixel_values": processed["pixel_values"][0],
                "attention_mask": processed["attention_mask"][0]
            }
        except Exception:
            # Skip missing/corrupt images
            return self.__getitem__((idx + 1) % len(self))

def setup_model():
    print(f"🧠 Loading {MODEL_NAME}...")

    # Fix for ambiguous return types
    base_model = CLIPModel.from_pretrained(MODEL_NAME)
    model = base_model.to(torch.device(device)) # type: ignore

    # Strict processor handling
    processor_raw = CLIPProcessor.from_pretrained(MODEL_NAME)
    if isinstance(processor_raw, tuple):
        processor_raw = processor_raw[0]
    processor = cast(CLIPProcessor, processor_raw)

    config = LoraConfig(
        r=16, lora_alpha=32, target_modules=["q_proj", "v_proj"],
        lora_dropout=0.05, bias="none"
    )

    # Wrap model with PEFT
    peft_model = get_peft_model(model, config)
    peft_model.print_trainable_parameters()

    # Explicit cast to stop Pylance complaining about .to() on PeftModel
    peft_model = peft_model.to(torch.device(device)) # type: ignore

    return peft_model, processor

def main():
    model, processor = setup_model()

    # Check if images exist before starting
    if not os.path.exists(IMAGE_ROOT):
        print(f"❌ Error: Image folder not found at {IMAGE_ROOT}")
        print("   Did you run the 'mv' commands to merge train-part1 and train-part2?")
        return

    dataset = FlatPhotoDataset(IMAGE_ROOT, METADATA_DIR, processor)

    if len(dataset) == 0:
        print("❌ Error: No images found. Check your paths.")
        return

    dataloader = DataLoader(dataset, batch_size=BATCH_SIZE, shuffle=True, num_workers=0)

    # FIX 1: Using the explicitly imported AdamW
    optimizer = AdamW(model.parameters(), lr=LEARNING_RATE)

    print("🚀 Starting Training...")
    model.train()

    for epoch in range(EPOCHS):
        total_loss = 0
        pbar = tqdm(dataloader, desc=f"Epoch {epoch+1}/{EPOCHS}")

        for batch in pbar:
            # Fix strict device typing
            batch = {k: v.to(torch.device(device)) for k, v in batch.items()}

            optimizer.zero_grad()
            outputs = model(**batch, return_loss=True)
            loss = outputs.loss
            loss.backward()
            optimizer.step()
            total_loss += loss.item()
            pbar.set_postfix(loss=loss.item())

        print(f"📉 Epoch {epoch+1} Loss: {total_loss / len(dataloader):.4f}")
        model.save_pretrained(f"fine_tuning/clip_finetuned_epoch_{epoch+1}")

if __name__ == "__main__":
    main()
