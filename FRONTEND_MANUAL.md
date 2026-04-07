# Frontend Integration Spec: AI Photo Retrieval System
**Version:** 2.0 (Multi-Tenant & Live Sync Update)
**Base API URL:** `http://13.61.195.243`

This document outlines the REST API endpoints mapped to UI actions, the required payloads, and the real-time Server-Sent Events (SSE) logic for updating the UI state.

> ⚠️ **CRITICAL ARCHITECTURE NOTE:**
> The system is now strictly Multi-Tenant. **Every single request** must include the `user_id`. Do not rely on backend defaults. If `user_id` is missing, the API will return a `400 Bad Request`.

---

## 🔘 PART 1: Core Endpoints & UI Mapping

Map these endpoints to the corresponding buttons/actions in the frontend UI.

### 1. Upload Images ("Upload" / "Add to Gallery" Button)
* **Endpoint:** `POST /api/upload/image`
* **Content-Type:** `multipart/form-data`
* **Payload:** * `user_id` (integer)
  * `files[]` (file array)
  * `dedup` (boolean, set to `true`)
* **Behavior:** Returns an array of uploaded items with their new `id` and `url`. The UI should immediately render these as "Grey / Processing" placeholders.

### 2. Search by Text ("Search" Bar)
* **Endpoint:** `POST /api/search/text`
* **Content-Type:** `application/json`
* **Payload:** `{"user_id": 1, "q": "white flowers", "limit": 10}`
* **Response:** Returns an array of image objects and a `used_qwen` boolean. (If `used_qwen` is false, UI can optionally show a "Fast Search Used" badge).

### 3. Granular Delete ("Trash" Icon on a specific image)
* **Endpoint:** `DELETE /api/user/images/delete-batch`
* **Content-Type:** `application/json`
* **Payload:** `{"user_id": 1, "image_ids": [42, 43]}`
* **Behavior:** Removes the specific images from the UI state.

### 4. Clear Gallery ("Empty Gallery" Button in Settings)
* **Endpoint:** `DELETE /api/user/images/clear`
* **Content-Type:** `application/json`
* **Payload:** `{"user_id": 1}`
* **Behavior:** Wipes all images for the user but keeps their account active. Clear the frontend gallery view.

### 5. Nuke Account ("Delete Account" Button in Settings)
* **Endpoint:** `DELETE /api/user/account`
* **Content-Type:** `application/json`
* **Payload:** `{"user_id": 1}`
* **Behavior:** Irreversibly destroys all user data. Redirect to login/signup screen.

---

## 📡 PART 2: Real-Time Status Streaming (SSE)

The frontend does **not** need to poll the backend to know when images are processed. The backend pushes live updates via Server-Sent Events (SSE) connected to a Redis Pub/Sub queue.

### Connection Setup
When the user logs in or opens the gallery, establish an `EventSource` connection to:
`GET /api/status-stream`

### Incoming Payload Format
When a background worker finishes processing an image, the stream will emit a JSON object:
```json
{
  "media_id": 42,
  "user_id": 1,
  "status": "fast_encoded" 
}
```
*(Possible statuses: `pending`, `fast_encoded`, `slow_encoded`, `in_index`, `failed`)*

---

## 🚥 PART 3: UI/UX State Logic (The Traffic Light System)

Use the incoming SSE payloads to drive the visual state of the gallery. 

### Item-Level Markers (Per Image)
Each image component should have a visual indicator (a border, a dot, or an overlay) based on its current state in the Redux/Zustand store:
* ⚪ **Grey (Unprocessed):** Initial state upon upload. The image is in S3 but not yet searchable.
* 🟡 **Yellow (Fast / SigLIP):** Receives `fast_encoded`. Image is now searchable via basic similarity, but lacks deep contextual AI understanding.
* 🟢 **Green (Slow / Qwen):** Receives `slow_encoded` or `in_index`. Image is fully processed, highly searchable, and contextualized.
* 🔴 **Red (Failed):** Receives `failed`. The ML worker crashed on this image (e.g., corrupted file). Show a retry/delete prompt.

### Global Gallery Marker (Top Navigation Bar)
To let the user know what "level" of search is currently available for their whole gallery, implement a Global Status Indicator based on the **lowest common denominator** of their uploaded items:

1. **If ANY single image is Grey (⚪):** * Global Marker = **Grey**. 
   * *Tooltip:* "Indexing new images... search results may be incomplete."
2. **If NO images are Grey, but ANY single image is Yellow (🟡):** * Global Marker = **Yellow**. 
   * *Tooltip:* "Basic search active. Deep AI contextualization in progress..."
3. **If ALL images are Green (🟢):** * Global Marker = **Green**. 
   * *Tooltip:* "Gallery fully synchronized. Deep AI search active."