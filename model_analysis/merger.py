import json
from pathlib import Path

# Paths
BASE_DIR = Path(__file__).parent
EVAL_DIR = BASE_DIR / "evaluation_dataset" / "elnaz"
OUTPUT_FILE = BASE_DIR / "elnaz_master_metadata.json"

def merge_metadata():
    master_metadata = {}
    
    # Find all metadata.json files in elnaz's subfolders
    meta_files = sorted(list(EVAL_DIR.glob("batch_*/sub_*/metadata.json")))
    
    print(f"📦 Found {len(meta_files)} sub-batch files. Merging...")

    for meta_file in meta_files:
        with open(meta_file, 'r') as f:
            try:
                data = json.load(f)
                # The keys are filenames, values are {'description': ..., 'tags': [...]}
                master_metadata.update(data)
            except json.JSONDecodeError:
                print(f"⚠️ Error reading {meta_file}")

    with open(OUTPUT_FILE, 'w') as f:
        json.dump(master_metadata, f, indent=4)

    print(f"✅ Successfully merged {len(master_metadata)} entries into {OUTPUT_FILE.name}")

if __name__ == "__main__":
    merge_metadata()
