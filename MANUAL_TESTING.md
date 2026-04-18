# AI Photo Retrieval System - Manual Testing Log
**Date:** April 18, 2026
**Environment:** AWS EC2 (Backend Gateway) + Vast.ai (GPU Workers) + AWS S3 + Milvus + PostgreSQL + Redis

This document serves as the official manual testing record for the **Production-Grade Batch Upload Pipeline** and the **Semantic Vector Search** capabilities. All tests were executed to verify the integration between the Go backend and the remote GPU inference workers.

---

## 🧪 Phase 1: Robust Batch Upload & Deduplication

**Objective:** Verify that the system can handle bulk uploads using the optimized bash-array method and correctly identify duplicate files based on content hashes.

### 1. The "Proper" Batch Upload
* **Action:** Executed a bash script to collect 13 unique images and their corresponding local Android paths into a single multipart request.
* **Result:** Successfully uploaded all 13 images. Database assigned IDs 1–13. Status correctly set to `active`.
* **Bash Command:**
```bash
# Prepare the multipart request array
unset args
args=(
  -s -X POST http://13.61.195.243/api/upload/image
  -H "Accept: application/json"
  -F "user_id=42"
  -F "dedup=true"
)

# Append files and local paths
for file in *.jpg; do
  args+=("-F" "files[]=@$file")
  args+=("-F" "local_paths[]=/storage/emulated/0/DCIM/$file")
done

# Execute
curl "${args[@]}" | jq
```

### 2. Content-Based Deduplication
* **Action:** Re-ran the same upload script immediately.
* **Result:** The API returned `"status": "duplicate"` for all 13 files. The system correctly cross-referenced the `user_id` and `checksum` to prevent redundant storage. **PASS**.

---

## 🧪 Phase 2: Semantic Vector Search Verification

**Objective:** Verify that the SigLIP model correctly maps text queries to images in vector space, specifically testing for synonyms and conceptual understanding rather than simple filename matching.

### 1. Exact Match Search
* **Action:** Search for "laptop" (filename was `laptop.jpg`).
* **Result:** Returned the correct `local_path` with high similarity score.
* **Command:**
```bash
curl -s -X POST http://13.61.195.243/api/search/text \
  -H "Content-Type: application/json" \
  -d '{"user_id": 42, "q": "laptop", "limit": 1}' | jq -r '.results[0].local_path'
```

### 2. Synonym & Conceptual Search (The "Synonym Test")
* **Action:** Tested the engine with words not present in the filenames to verify semantic understanding.
* **Test Matrix:**

| Query | Target Image | Result |
| :--- | :--- | :--- |
| `arm` | `hand.jpg` | **SUCCESS** |
| `spectacles` | `glasses.jpg` | **SUCCESS** |
| `footwear` | `slippers.jpg` | **SUCCESS** |
| `clutter` | `junk on the desk.jpg` | **SUCCESS** |
| `power adapter` | `a charger.jpg` | **SUCCESS** |

---

## 🧪 Phase 3: Multi-Tier Deletion Logic

**Objective:** Verify the surgical removal of data across the three layers (Postgres, S3, and Milvus).

### 1. Batch ID Deletion
* **Action:** Delete a specific set of Image IDs.
* **Command:**
```bash
curl -X DELETE http://13.61.195.243/api/user/images/delete-batch \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 42,
    "image_ids": [1, 2, 3]
  }'
```

### 2. Gallery Purge (Clear All)
* **Action:** Wipe the entire gallery for a user without deleting the user account.
* **Command:**
```bash
curl -X DELETE http://13.61.195.243/api/user/images/clear \
  -H "Content-Type: application/json" \
  -d '{"user_id": 42}'
```

---

## 🛠️ Infrastructure & Debugging Notes

1.  **GPU Scheduling:** Remote GPU workers on Vast.ai are "interruptible." If a `connection refused` error occurs, the worker instance must be destroyed and a new one rented, with the `ML_SERVICE_URL` updated in the `.env` file.
2.  **Schema Enforcement:** Manual schema application via `apply_migrations.sql` is required if the backend container lacks the `goose` migration tool. 
    * **Fix Applied:** `vec_bytes` set to `Nullable` to allow `ClaimForProcessing` inserts.
    * **Fix Applied:** `status` CHECK constraint updated to include `fast_encoded` and `slow_encoded`.
3.  **API Schema:** The text search endpoint requires the key **`"q"`** for the search string. The response returns results in the format `.results[].local_path`.

---

## 🏆 Final Status: **VERIFIED**
The system successfully processes bulk uploads, handles asynchronous inference on remote hardware, and retrieves images based on complex semantic concepts. The pipeline is stable.
