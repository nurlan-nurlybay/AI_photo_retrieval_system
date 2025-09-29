import torchvision
import torchvision.transforms as T
from PIL import Image
import os

# Where to save images
OUT_DIR = "assets/cifar10"
os.makedirs(OUT_DIR, exist_ok=True)

# Download CIFAR-10
dataset = torchvision.datasets.CIFAR10(
    root="./data",
    train=True,
    download=True
)


# Save as PNGs grouped by class
for idx, (img, label) in enumerate(dataset):
    label_name = dataset.classes[label]
    class_dir = os.path.join(OUT_DIR, label_name)
    os.makedirs(class_dir, exist_ok=True)
    img.save(os.path.join(class_dir, f"{label_name}_{idx}.png"))
    if idx >= 499:  # stop at 500 if you only want a small seed set
        break

print("Saved PNGs to", OUT_DIR)
