import 'dart:async';
import 'dart:io';

import 'package:photo_manager/photo_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:frontend_flutter/data/api_client.dart';

// ---------------------------------------------------------------------------
// Progress model
// ---------------------------------------------------------------------------

enum PhotoSyncStatus { idle, checking, syncing, completed, error }

class PhotoSyncProgress {
  final PhotoSyncStatus status;
  final int totalPhotos;
  final int syncedPhotos;
  final int failedPhotos;
  final String? errorMessage;

  const PhotoSyncProgress({
    this.status = PhotoSyncStatus.idle,
    this.totalPhotos = 0,
    this.syncedPhotos = 0,
    this.failedPhotos = 0,
    this.errorMessage,
  });

  String get displayText {
    switch (status) {
      case PhotoSyncStatus.idle:
        return '';
      case PhotoSyncStatus.checking:
        return 'Checking...';
      case PhotoSyncStatus.syncing:
        return 'Syncing $syncedPhotos/$totalPhotos photos...';
      case PhotoSyncStatus.completed:
        return 'Synced $syncedPhotos photos';
      case PhotoSyncStatus.error:
        return errorMessage ?? 'Sync failed';
    }
  }

  double get progressFraction =>
      totalPhotos > 0 ? syncedPhotos / totalPhotos : 0;
}

// ---------------------------------------------------------------------------
// Singleton service
// ---------------------------------------------------------------------------

class PhotoSyncService {
  PhotoSyncService._();
  static final PhotoSyncService instance = PhotoSyncService._();

  static const String _prefSyncedIds = 'photo_sync_synced_ids';
  static const String _prefLastRun = 'photo_sync_last_run_ms';
  static const String prefNeedsCategoryRefresh = 'needs_category_refresh';
  static const int _batchSize = 10;

  final ApiClient _api = ApiClient();
  bool _isSyncing = false;

  final StreamController<PhotoSyncProgress> _controller =
      StreamController<PhotoSyncProgress>.broadcast();

  Stream<PhotoSyncProgress> get progressStream => _controller.stream;

  PhotoSyncProgress _lastProgress = const PhotoSyncProgress();
  PhotoSyncProgress get lastProgress => _lastProgress;

  void _emit(PhotoSyncProgress p) {
    _lastProgress = p;
    _controller.add(p);
  }

  // -----------------------------------------------------------------------
  // Public API
  // -----------------------------------------------------------------------

  /// Call after any manual upload (outside of auto-sync) so that
  /// CategoriesScreen and other listeners know new images were added.
  Future<void> notifyNewUploads(int count) async {
    if (count <= 0) return;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(prefNeedsCategoryRefresh, true);
    _emit(PhotoSyncProgress(
      status: PhotoSyncStatus.completed,
      syncedPhotos: count,
      totalPhotos: count,
    ));
  }

  Future<void> startSync() async {
    if (_isSyncing) return;
    _isSyncing = true;

    try {
      // 1. Checking phase
      _emit(const PhotoSyncProgress(status: PhotoSyncStatus.checking));

      // 2. Health check
      final isOnline = await _api.checkHealth();
      if (!isOnline) {
        _emit(const PhotoSyncProgress(
          status: PhotoSyncStatus.error,
          errorMessage: 'Backend offline',
        ));
        return;
      }

      // 3. Gallery permission
      final perm = await PhotoManager.requestPermissionExtend();
      if (!perm.isAuth) {
        _emit(const PhotoSyncProgress(
          status: PhotoSyncStatus.error,
          errorMessage: 'Gallery permission denied',
        ));
        return;
      }

      // 4. Load ALL device photos (paginated, newest first)
      final allAssets = await _loadAllAssets();
      if (allAssets.isEmpty) {
        _emit(const PhotoSyncProgress(status: PhotoSyncStatus.completed));
        return;
      }

      // 5. Load already-synced IDs from SharedPreferences
      final prefs = await SharedPreferences.getInstance();
      final syncedIds =
          (prefs.getStringList(_prefSyncedIds) ?? []).toSet();

      // 6. Filter unsynced
      final unsynced =
          allAssets.where((a) => !syncedIds.contains(a.id)).toList();

      if (unsynced.isEmpty) {
        _emit(PhotoSyncProgress(
          status: PhotoSyncStatus.completed,
          totalPhotos: allAssets.length,
          syncedPhotos: allAssets.length,
        ));
        return;
      }

      // 7. Upload in batches
      final totalToSync = unsynced.length;
      int synced = 0;
      int failed = 0;

      _emit(PhotoSyncProgress(
        status: PhotoSyncStatus.syncing,
        totalPhotos: totalToSync,
      ));

      for (int i = 0; i < unsynced.length; i += _batchSize) {
        final batchEnd =
            (i + _batchSize > unsynced.length) ? unsynced.length : i + _batchSize;
        final batch = unsynced.sublist(i, batchEnd);

        try {
          final result = await _uploadBatch(batch);
          synced += result.synced;
          failed += result.failed;

          // 8. Persist after each batch (crash-safe)
          syncedIds.addAll(result.syncedAssetIds);
          await prefs.setStringList(_prefSyncedIds, syncedIds.toList());
        } catch (_) {
          // Failed batches don't abort sync
          failed += batch.length;
        }

        _emit(PhotoSyncProgress(
          status: PhotoSyncStatus.syncing,
          totalPhotos: totalToSync,
          syncedPhotos: synced,
          failedPhotos: failed,
        ));
      }

      // 9. Record last run timestamp
      await prefs.setInt(_prefLastRun, DateTime.now().millisecondsSinceEpoch);
      if (synced > 0) {
        await prefs.setBool(prefNeedsCategoryRefresh, true);
      }

      _emit(PhotoSyncProgress(
        status: PhotoSyncStatus.completed,
        totalPhotos: totalToSync,
        syncedPhotos: synced,
        failedPhotos: failed,
      ));
    } catch (e) {
      _emit(PhotoSyncProgress(
        status: PhotoSyncStatus.error,
        errorMessage: e.toString(),
      ));
    } finally {
      _isSyncing = false;
    }
  }

  // -----------------------------------------------------------------------
  // Internals
  // -----------------------------------------------------------------------

  /// Paginate through all device photos (newest first), 100 per page.
  Future<List<AssetEntity>> _loadAllAssets() async {
    final albums = await PhotoManager.getAssetPathList(
      type: RequestType.image,
      filterOption: FilterOptionGroup(
        orders: [
          const OrderOption(type: OrderOptionType.createDate, asc: false),
        ],
      ),
    );
    if (albums.isEmpty) return [];

    final recentAlbum = albums.first;
    final totalCount = await recentAlbum.assetCountAsync;

    final List<AssetEntity> all = [];
    const pageSize = 100;

    for (int page = 0; page * pageSize < totalCount; page++) {
      final start = page * pageSize;
      final end = start + pageSize;
      final assets = await recentAlbum.getAssetListRange(
        start: start,
        end: end > totalCount ? totalCount : end,
      );
      all.addAll(assets);
    }

    return all;
  }

  /// Upload a single batch and return per-file results.
  Future<_BatchResult> _uploadBatch(List<AssetEntity> batch) async {
    final List<File> files = [];
    final List<String> localPaths = [];
    final List<String> assetIds = [];

    for (final asset in batch) {
      final file = await asset.file;
      if (file == null) continue; // iCloud photo not available — skip
      files.add(file);
      localPaths.add(file.path);
      assetIds.add(asset.id);
    }

    if (files.isEmpty) {
      return _BatchResult(synced: 0, failed: 0, syncedAssetIds: []);
    }

    final resp =
        await _api.uploadImages(files: files, localPaths: localPaths, dedup: true);

    int synced = 0;
    int failed = 0;
    final List<String> syncedAssetIds = [];

    for (int j = 0; j < resp.results.length; j++) {
      final r = resp.results[j];
      if (r.status == 'saved' || r.status == 'duplicate') {
        synced++;
        if (j < assetIds.length) syncedAssetIds.add(assetIds[j]);
      } else {
        failed++;
      }
    }

    // Also mark assets whose file was skipped (iCloud) as not synced — they
    // aren't in assetIds so nothing extra needed.

    return _BatchResult(
      synced: synced,
      failed: failed,
      syncedAssetIds: syncedAssetIds,
    );
  }
}

class _BatchResult {
  final int synced;
  final int failed;
  final List<String> syncedAssetIds;

  _BatchResult({
    required this.synced,
    required this.failed,
    required this.syncedAssetIds,
  });
}
