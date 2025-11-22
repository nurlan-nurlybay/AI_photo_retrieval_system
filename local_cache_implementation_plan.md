# Implementation Plan: Local Image Cache with Fallback

## Overview
Enable the app to display images from local device storage when available, falling back to SeaweedFS only when necessary. This reduces bandwidth and improves performance.

---

## Phase 1: Database Schema Changes

### Backend: PostgreSQL `media` Table
**File:** `backend/migrations/000X_add_local_path.sql`

Add a new column to store the device-local path:
```sql
ALTER TABLE media ADD COLUMN local_path TEXT;
CREATE INDEX idx_media_local_path ON media(user_id, local_path);
```

**Considerations:**
- `local_path` should be nullable (images uploaded from web won't have it)
- Store the full path or just the relative path from a known root
- Consider privacy: paths may contain user info

---

## Phase 2: Backend API Changes

### 2.1 Upload Endpoint Modification
**File:** `backend/internal/adapter/http/dto/httpdto.go`

Update `UploadRequest` to accept local path:
```go
type UploadRequest struct {
    Files      []*multipart.FileHeader `form:"files[]" binding:"required"`
    LocalPaths []string                `form:"local_paths[]"` // NEW: parallel array
    Dedup      bool                    `form:"dedup,default=true"`
}
```

**File:** `backend/internal/adapter/http/upload_handler.go`

Modify `ImageUpload` handler:
- Extract `local_paths[]` from form data
- Pass local path to `UploadInput` in usecase layer

### 2.2 Domain Model Update
**File:** `backend/internal/domain/media.go`

Add field to `Media` struct:
```go
type Media struct {
    // ... existing fields
    LocalPath string // NEW
}
```

### 2.3 Repository Layer
**File:** `backend/internal/adapter/postgres/media_repo.go`

Update SQL queries:
- `Create`: Include `local_path` in INSERT
- `Get`, `List`: Include `local_path` in SELECT
- `GetByChecksum`: Include `local_path` in SELECT

### 2.4 Response DTOs
**File:** `backend/internal/adapter/http/dto/httpdto.go`

Update response structs:
```go
type MediaItem struct {
    // ... existing fields
    LocalPath string `json:"local_path,omitempty"` // NEW
}

type MediaResponse struct {
    // ... existing fields
    LocalPath string `json:"local_path,omitempty"` // NEW
}
```

**Files to update:**
- `upload_handler.go`: Map `LocalPath` in responses
- `search_handler.go`: Map `LocalPath` in search results

---

## Phase 3: Frontend Data Layer Changes

### 3.1 Models Update
**File:** `frontend_flutter/lib/data/models.dart`

Add `localPath` to all media models:
```dart
class MediaItem {
  final int id;
  final String url;
  final String thumbUrl;
  final String? localPath;  // NEW
  // ... other fields
}

class MediaResponse {
  final int id;
  final String url;
  final String thumbUrl;
  final String? localPath;  // NEW
  // ... other fields
}
```

Update `fromJson` constructors to parse `local_path`.

### 3.2 API Client Update
**File:** `frontend_flutter/lib/data/api_client.dart`

#### Upload Method
Modify `uploadImages` to send local paths:
```dart
static Future<UploadResponseModel> uploadImages({
  required List<File> files,
  required List<String> localPaths,  // NEW: must match files length
}) async {
  var request = http.MultipartRequest('POST', uri);
  
  // Add files
  for (var file in files) {
    request.files.add(await http.MultipartFile.fromPath('files[]', file.path));
  }
  
  // Add local paths (NEW)
  for (var path in localPaths) {
    request.fields['local_paths[]'] = path;
  }
}
```

**Note:** Ensure `localPaths` array is in the same order as `files`.

---

## Phase 4: Frontend UI Layer Changes

### 4.1 Image Widget Helper
**File:** `frontend_flutter/lib/ui/widgets/cached_image.dart` (NEW)

Create a smart image widget:
```dart
class CachedImage extends StatelessWidget {
  final String? localPath;
  final String remoteUrl;
  final BoxFit fit;
  
  Widget build(BuildContext context) {
    // 1. Check if localPath exists and file is accessible
    if (localPath != null && File(localPath).existsSync()) {
      return Image.file(File(localPath), fit: fit);
    }
    
    // 2. Fallback to network image
    return Image.network(AppConfig.fixImageUrl(remoteUrl), fit: fit);
  }
}
```

**Considerations:**
- Add error handling for `File.existsSync()` (permissions)
- Consider async check to avoid blocking UI
- Add loading/error states

### 4.2 Home Screen Update
**File:** `frontend_flutter/lib/ui/screens/home_screen.dart`

#### Upload Flow
Modify `_uploadNewImages`:
```dart
Future<void> _uploadNewImages() async {
  final result = await FilePicker.platform.pickFiles(allowMultiple: true);
  
  List<File> files = [];
  List<String> localPaths = [];  // NEW
  
  for (var path in result.paths) {
    files.add(File(path));
    localPaths.add(path);  // Store original path
  }
  
  await ApiClient.uploadImages(
    files: files,
    localPaths: localPaths,  // NEW
  );
}
```

#### Display Flow
Replace `Image.network` with `CachedImage`:
```dart
// OLD:
Image.network(AppConfig.fixImageUrl(item.thumbUrl))

// NEW:
CachedImage(
  localPath: item.localPath,
  remoteUrl: item.thumbUrl,
  fit: BoxFit.cover,
)
```

### 4.3 Detail Screen Update
**File:** `frontend_flutter/lib/ui/screens/detail_screen.dart`

Same pattern - replace `Image.network` with `CachedImage`:
```dart
CachedImage(
  localPath: mediaItem?.localPath,
  remoteUrl: imageUrl,
  fit: BoxFit.contain,
)
```

---

## Phase 5: Android Permissions

### 5.1 Manifest Update
**File:** `frontend_flutter/android/app/src/main/AndroidManifest.xml`

Add storage permissions:
```xml
<uses-permission android:name="android.permission.READ_EXTERNAL_STORAGE"/>
<uses-permission android:name="android.permission.READ_MEDIA_IMAGES"/> <!-- Android 13+ -->
```

### 5.2 Runtime Permission Handling
**File:** `frontend_flutter/lib/ui/screens/home_screen.dart`

Request permissions before accessing local files:
```dart
// Add permission_handler package to pubspec.yaml
import 'package:permission_handler/permission.dart';

Future<void> _checkStoragePermission() async {
  if (await Permission.photos.isDenied) {
    await Permission.photos.request();
  }
}
```

Call `_checkStoragePermission()` in `initState()`.

---

## Phase 6: Edge Cases & Considerations

### 6.1 Path Persistence Issues
**Problem:** Local paths may become invalid if:
- User moves/deletes the image
- App is reinstalled
- Storage is unmounted

**Solution:**
- Always check `File.existsSync()` before using local path
- Gracefully fallback to remote URL
- Consider a background job to validate paths periodically

### 6.2 Multi-Device Sync
**Problem:** User uploads from Phone A, views on Phone B
- Phone B doesn't have the local file

**Solution:**
- `local_path` is device-specific, not synced
- Phone B will automatically use remote URL
- Consider storing device ID with local path if needed

### 6.3 Privacy Concerns
**Problem:** Local paths may expose user directory structure

**Solution:**
- Store only relative paths (e.g., from `DCIM/Camera/`)
- Or hash the path for privacy
- Ensure backend doesn't log full paths

### 6.4 Thumbnail vs Original
**Decision needed:**
- Should `local_path` point to original or thumbnail?
- Recommendation: Point to original, generate thumbnail on-device if needed

---

## Phase 7: Testing Checklist

### Backend Tests
- [ ] Upload with `local_paths[]` saves to database
- [ ] Upload without `local_paths[]` still works (backward compatibility)
- [ ] Search results include `local_path` field
- [ ] List results include `local_path` field

### Frontend Tests
- [ ] Upload sends local paths correctly
- [ ] Images display from local file when available
- [ ] Images fallback to network when local file missing
- [ ] Permissions are requested on Android
- [ ] Works on both Android and iOS (iOS paths differ)

### Integration Tests
- [ ] Upload → List → Display uses local cache
- [ ] Search results use local cache when available
- [ ] Multi-user scenario (different devices)

---

## Summary of Files to Modify

### Backend (Go)
1. `backend/migrations/000X_add_local_path.sql` - NEW
2. `backend/internal/domain/media.go` - Add `LocalPath` field
3. `backend/internal/adapter/postgres/media_repo.go` - Update SQL queries
4. `backend/internal/adapter/http/dto/httpdto.go` - Update DTOs
5. `backend/internal/adapter/http/upload_handler.go` - Parse `local_paths[]`
6. `backend/internal/adapter/http/search_handler.go` - Include `local_path` in responses
7. `backend/internal/usecase/uploadUC.go` - Pass local path through
8. `backend/internal/usecase/searchUC.go` - Include local path in results

### Frontend (Flutter)
1. `frontend_flutter/pubspec.yaml` - Add `permission_handler` dependency
2. `frontend_flutter/lib/data/models.dart` - Add `localPath` to models
3. `frontend_flutter/lib/data/api_client.dart` - Send `local_paths[]` in upload
4. `frontend_flutter/lib/ui/widgets/cached_image.dart` - NEW smart image widget
5. `frontend_flutter/lib/ui/screens/home_screen.dart` - Use `CachedImage`, send paths
6. `frontend_flutter/lib/ui/screens/detail_screen.dart` - Use `CachedImage`
7. `frontend_flutter/android/app/src/main/AndroidManifest.xml` - Add permissions

### Total: ~15 files to modify/create

---

## Recommended Implementation Order

1. **Backend schema** (migration) - Foundation
2. **Backend domain/repo** - Data layer
3. **Backend DTOs/handlers** - API layer
4. **Frontend models** - Data structures
5. **Frontend API client** - Network layer
6. **Frontend CachedImage widget** - Reusable component
7. **Frontend UI updates** - User-facing changes
8. **Permissions** - Android-specific
9. **Testing** - Validation

**Estimated effort:** 4-6 hours for a single developer
