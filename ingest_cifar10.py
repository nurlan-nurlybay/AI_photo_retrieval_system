import requests
import torchvision
import torchvision.transforms as T
from io import BytesIO
from PIL import Image
import psycopg2

SEAWEED_MASTER = "http://localhost:9333"
FAISS_URL = "http://localhost:8002/v1/vectors/add"
CLIP_URL = "http://localhost:8003/v1/encode/image"

# CIFAR-10
transform = T.ToTensor()
dataset = torchvision.datasets.CIFAR10(root="./data", train=True, download=True, transform=transform)
classes = dataset.classes


conn = psycopg2.connect(
    dbname="media",
    user="postgres",
    password="postgres",
    host="127.0.0.1",   # host network
    port=5432,
)
cur = conn.cursor()

def save_to_db(img_id: str, url: str, label: str):
    cur.execute(
        "INSERT INTO images (id, url, label) VALUES (%s, %s, %s) ON CONFLICT (id) DO NOTHING",
        (img_id, url, label)
    )
    conn.commit()

def upload_to_seaweedfs(img: Image.Image) -> str:
    print("[seaweed] requesting fid from master…")
    assign = requests.get(f"{SEAWEED_MASTER}/dir/assign").json()
    fid = assign["fid"]
    public_url = assign["publicUrl"]
    print(f"[seaweed] got fid={fid}, publicUrl={public_url}")

    # Image metadata
    print(f"[cifar] image mode={img.mode}, size={img.size}")

    buf = BytesIO()
    img.save(buf, format="PNG")
    size_bytes = buf.tell()
    buf.seek(0)
    print(f"[cifar] PNG bytes length={size_bytes}")

    files = {"file": ("img.png", buf, "image/png")}
    target_url = f"http://{public_url}/{fid}"
    print(f"[seaweed] uploading to {target_url}")
    resp = requests.post(target_url, files=files)
    print(f"[seaweed] upload response: {resp.status_code}")
    resp.raise_for_status()
    return target_url



def encode_image_remote(img: Image.Image) -> list[float]:
    print("[clip] encoding image…")
    buf = BytesIO()
    img.save(buf, format="PNG")
    buf.seek(0)
    files = {"file": ("img.png", buf, "image/png")}
    resp = requests.post(CLIP_URL, files=files)
    print(f"[clip] response: {resp.status_code}")
    resp.raise_for_status()
    return resp.json()["vector"]


def insert_faiss(img_id: str, vector: list[float]):
    print(f"[faiss] inserting {img_id}")
    payload = {"id": img_id, "vector": vector}
    resp = requests.post(FAISS_URL, json=payload)
    print(f"[faiss] response: {resp.status_code}")
    resp.raise_for_status()


def main(n=100):
    for idx in range(n):
        img, label = dataset[idx]
        pil_img = T.ToPILImage()(img)
        class_name = classes[label]
        img_id = f"cifar_{idx}_{class_name}"

        print(f"\n=== processing {img_id} ===")
        url = upload_to_seaweedfs(pil_img)
        vec = encode_image_remote(pil_img)
        insert_faiss(img_id, vec)
        print(f"[done] {img_id} → {url}")

        save_to_db(img_id, url, class_name)



if __name__ == "__main__":
    main(n=500)
