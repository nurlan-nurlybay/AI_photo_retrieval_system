import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:photo_manager/photo_manager.dart';

import 'package:shared_preferences/shared_preferences.dart';

import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/core/photo_sync.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/screens/detail_screen.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';
import 'package:frontend_flutter/core/status_stream_service.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final TextEditingController _searchController = TextEditingController();
  final ImagePicker _picker = ImagePicker();
  final ApiClient _api = ApiClient();

  final Duration _debounceDuration = const Duration(milliseconds: 450);

  Timer? _debounce;
  bool _isLoading = false;
  bool _isSearchMode = false;
  bool _isBackendOnline = false;
  bool _isWakingUp = false; // For cold start detection
  String? _error;

  StreamSubscription<StatusUpdate>? _statusSub;
  ImageProcessingStatus _globalStatus = ImageProcessingStatus.inIndex;

  /// Device gallery assets (shown in gallery mode).
  List<AssetEntity> _galleryAssets = [];

  /// Set of local file paths that have been synced to the backend.
  Set<String> _syncedIds = {};

  /// Backend search results (shown in search mode).
  List<MediaItem> _searchResults = [];

  @override
  void initState() {
    super.initState();
    _checkBackendStatus();
    _loadGalleryPhotos();
    _loadSyncedIds();
    _triggerAutoSync();
    _initStatusStream();
  }

  Future<void> _loadSyncedIds() async {
    final prefs = await SharedPreferences.getInstance();
    final ids = prefs.getStringList('photo_sync_synced_ids') ?? [];
    if (mounted) setState(() => _syncedIds = ids.toSet());
  }

  void _initStatusStream() {
    print('[${_ts()}] 📡 [SSE] Starting StatusStreamService...');
    StatusStreamService.instance.start();
    _statusSub = StatusStreamService.instance.stream.listen((update) {
      if (!mounted) return;
      print('[${_ts()}] 📡 [SSE] Received update: mediaId=${update.mediaId} status=${update.status.name}');
      setState(() {
        final index = _searchResults.indexWhere((item) => item.id == update.mediaId);
        if (index != -1) {
          print('[${_ts()}] 📡 [SSE] Updated search result [$index] status -> ${update.status.name}');
          _searchResults[index].status = update.status;
        }
        _updateGlobalStatus();
      });
    });
  }

  static String _ts() => DateTime.now().toIso8601String().substring(11, 23);

  void _updateGlobalStatus() {
    if (_searchResults.isEmpty) {
      // No results on screen — don't touch the badge, let SSE drive it.
      return;
    }
    // Best-case logic: only upgrade the badge, never downgrade to grey.
    // Search results default to status=unprocessed (backend omits the field),
    // so checking "any unprocessed" would always be true → always grey. Wrong.
    // We only promote to yellow/green once SSE has confirmed real processing.
    if (_searchResults.any((item) =>
        item.status == ImageProcessingStatus.inIndex ||
        item.status == ImageProcessingStatus.slowEncoded)) {
      _globalStatus = ImageProcessingStatus.inIndex;
    } else if (_searchResults.any((item) =>
        item.status == ImageProcessingStatus.fastEncoded)) {
      _globalStatus = ImageProcessingStatus.fastEncoded;
    }
    // All still at default unprocessed → leave badge unchanged.
  }

  Future<void> _triggerAutoSync() async {
    final isOnline = await _api.checkHealth();
    if (isOnline) {
      // If the backend has no photos for this user but SharedPreferences
      // thinks everything was already synced, we have stale sync state.
      // Clear it so the next startSync() uploads everything fresh.
      await _clearStaleSyncStateIfNeeded();
      PhotoSyncService.instance.startSync();
      // Reload synced IDs once sync completes so cloud badges update.
      PhotoSyncService.instance.progressStream
          .where((p) => p.status == PhotoSyncStatus.completed)
          .first
          .then((_) { if (mounted) _loadSyncedIds(); });
    }
  }

  Future<void> _clearStaleSyncStateIfNeeded() async {
    try {
      final backendItems = await ApiClient.listUserImages(
        userId: AppConfig.userId,
        limit: 1,
      );
      if (backendItems.isEmpty) {
        final prefs = await SharedPreferences.getInstance();
        final syncedIds = prefs.getStringList('photo_sync_synced_ids') ?? [];
        if (syncedIds.isNotEmpty) {
          print('[${_ts()}] 🔄 [SYNC] Backend empty but ${syncedIds.length} IDs cached — clearing stale sync state');
          await prefs.remove('photo_sync_synced_ids');
          if (mounted) setState(() => _syncedIds = {});
        }
      }
    } catch (_) {
      // Non-critical — ignore errors
    }
  }

  Future<void> _checkBackendStatus() async {
    final start = DateTime.now();
    final isOnline = await _api.checkHealth();
    final duration = DateTime.now().difference(start);

    if (mounted) {
      setState(() {
        _isBackendOnline = isOnline;
        // If response took > 1.5s, it's likely a cold start / waking up
        _isWakingUp = isOnline && duration.inMilliseconds > 1500;
      });
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    _debounce?.cancel();
    _statusSub?.cancel();
    StatusStreamService.instance.stop();
    _api.close();
    super.dispose();
  }

  // ================= DATA LOADING =================

  /// Load photos directly from the device gallery.
  Future<void> _loadGalleryPhotos() async {
    setState(() {
      _isLoading = true;
      _error = null;
      _isSearchMode = false;
    });

    try {
      final permission = await PhotoManager.requestPermissionExtend();
      if (!permission.isAuth) {
        setState(() {
          _error = 'Gallery permission denied. Please grant access in Settings.';
          _isLoading = false;
        });
        return;
      }

      final albums = await PhotoManager.getAssetPathList(
        type: RequestType.image,
        filterOption: FilterOptionGroup(
          orders: [const OrderOption(type: OrderOptionType.createDate, asc: false)],
        ),
      );

      if (albums.isEmpty) {
        setState(() {
          _galleryAssets = [];
          _isLoading = false;
        });
        return;
      }

      // "Recent" / "All Photos" is typically the first album
      final recentAlbum = albums.first;
      final assets = await recentAlbum.getAssetListRange(start: 0, end: 200);

      setState(() {
        _galleryAssets = assets;
      });
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  /// Text search using /api/search/text
  Future<void> _searchByText(String query) async {
    final trimmed = query.trim();
    print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] query="$trimmed"');

    if (trimmed.isEmpty) {
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] Empty query -> loading gallery');
      _loadGalleryPhotos();
      return;
    }

    if (trimmed.length < 2) {
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] Query too short (<2 chars), clearing results');
      setState(() {
        _searchResults = [];
        _error = null;
        _isSearchMode = true;
      });
      return;
    }

    setState(() {
      _isLoading = true;
      _error = null;
      _isSearchMode = true;
    });

    try {
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] Calling _api.textSearch()...');
      final sw = Stopwatch()..start();
      final SearchResponse resp = await _api.textSearch(trimmed, limit: 30);
      sw.stop();
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] API returned ${resp.results.length} raw results in ${sw.elapsedMilliseconds}ms');

      final filtered = _filterResults(resp.results);
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] After filtering: ${filtered.length} results');

      setState(() {
        _searchResults = filtered
            .map((r) => MediaItem(
                  id: r.id,
                  url: r.url,
                  thumbUrl: r.thumbUrl,
                  mimeType: 'image/jpeg',
                  sizeBytes: 0,
                  width: 0,
                  height: 0,
                  checksum: '',
                  createdAt: 'Unknown',
                  localPath: r.localPath,
                  // Vector DB only returns indexed items → at least fastEncoded.
                  // Default to fastEncoded so dots are yellow from the start.
                  status: r.status == ImageProcessingStatus.unprocessed
                      ? ImageProcessingStatus.fastEncoded
                      : r.status,
                ))
            .toList();
        _isBackendOnline = true;
        // Do NOT call _updateGlobalStatus() — defaults would flip badge to grey.
      });
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] ✅ UI updated with ${_searchResults.length} items');
    } catch (e, st) {
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] ❌ ERROR: $e');
      print('[${_ts()}] 🔍 [TEXT_SEARCH_FLOW] stacktrace: ${st.toString().split('\n').take(3).join(' | ')}');
      setState(() => _error = e.toString());
    } finally {
      setState(() => _isLoading = false);
    }
  }

  /// Image search using /api/search/image
  Future<void> _searchByImage([File? preSelectedFile]) async {
    print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] Start. preSelected=${preSelectedFile != null}');
    File? file;

    if (preSelectedFile != null) {
      file = preSelectedFile;
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] Using preselected file: ${file.path}');
    } else {
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] Opening image picker...');
      final XFile? picked =
          await _picker.pickImage(source: ImageSource.gallery);
      if (picked == null) {
        print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] User cancelled picker');
        return;
      }
      file = File(picked.path);
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] Picked: ${file.path}');
    }

    final fileExists = await file.exists();
    final fileSize = fileExists ? await file.length() : 0;
    print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] File exists=$fileExists, size=${(fileSize / 1024).toStringAsFixed(1)}KB');

    setState(() {
      _isLoading = true;
      _error = null;
      _isSearchMode = true;
    });

    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Searching by image...'),
          duration: Duration(seconds: 2),
        ),
      );
    }

    try {
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] Calling _api.imageSearch()...');
      final sw = Stopwatch()..start();
      final SearchResponse resp = await _api.imageSearch(file);
      sw.stop();
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] API returned ${resp.results.length} raw results in ${sw.elapsedMilliseconds}ms');

      final filtered = _filterResults(resp.results, threshold: 0.25);
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] After filtering: ${filtered.length} results');

      setState(() {
        _searchResults = filtered
            .map((r) => MediaItem(
                  id: r.id,
                  url: r.url,
                  thumbUrl: r.thumbUrl,
                  mimeType: 'image/jpeg',
                  sizeBytes: 0,
                  width: 0,
                  height: 0,
                  checksum: '',
                  createdAt: 'Unknown',
                  localPath: r.localPath,
                  status: r.status == ImageProcessingStatus.unprocessed
                      ? ImageProcessingStatus.fastEncoded
                      : r.status,
                ))
            .toList();
        _isBackendOnline = true;
      });
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] ✅ UI updated with ${_searchResults.length} items');
    } catch (e, st) {
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] ❌ ERROR: $e');
      print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] stacktrace: ${st.toString().split('\n').take(3).join(' | ')}');
      // A 500 from the server usually means the user's photo index is empty —
      // show an empty result grid instead of a red error screen.
      final msg = e.toString();
      if (msg.contains('500') || msg.contains('internal error')) {
        print('[${_ts()}] 🖼️ [IMG_SEARCH_FLOW] Treating server 500 as empty results');
        setState(() {
          _searchResults = [];
          _error = null;
        });
      } else {
        setState(() => _error = msg);
      }
    } finally {
      setState(() => _isLoading = false);
    }
  }

  /// Minimum cosine-similarity to be considered a real match.
  static const double _minScore = 0.22;

  List<MediaResponse> _filterResults(List<MediaResponse> results,
      {double threshold = 0.12}) {
    print('[${_ts()}] 🔬 [FILTER] Input: ${results.length} results, minScore=$_minScore, dropThreshold=$threshold');
    if (results.isEmpty) {
      print('[${_ts()}] 🔬 [FILTER] Empty input -> returning []');
      return [];
    }

    // Log all raw scores
    for (var i = 0; i < results.length; i++) {
      print('[${_ts()}] 🔬 [FILTER] raw[$i] id=${results[i].id} score=${results[i].score}');
    }

    results.sort((a, b) => (b.score ?? 0).compareTo(a.score ?? 0));

    // Null score means the backend returned 0.0 (omitted by Go omitempty) —
    // images not yet in Milvus index. Pass them through so the user still
    // sees their photos while embeddings are being generated.
    final aboveFloor = results
        .where((r) => r.score == null || r.score! >= _minScore)
        .toList();
    print('[${_ts()}] 🔬 [FILTER] After floor filter: ${aboveFloor.length} (removed ${results.length - aboveFloor.length})');
    if (aboveFloor.isEmpty) return [];

    List<MediaResponse> kept = [aboveFloor[0]];
    for (int i = 1; i < aboveFloor.length; i++) {
      // Treat null score as 0.0 for the gap-drop check.
      final prevScore = aboveFloor[i - 1].score ?? 0.0;
      final currScore = aboveFloor[i].score ?? 0.0;
      final drop = prevScore - currScore;
      if (drop > threshold) {
        print('[${_ts()}] 🔬 [FILTER] Drop cutoff at [$i]: drop=$drop > threshold=$threshold');
        break;
      }
      kept.add(aboveFloor[i]);
    }
    print('[${_ts()}] 🔬 [FILTER] Final output: ${kept.length} results');
    return kept;
  }

  /// FAB upload – pick images, upload to backend, then refresh gallery.
  Future<void> _uploadNewImages() async {
    final List<XFile> picked = await _picker.pickMultiImage();
    if (picked.isEmpty) return;

    final files = picked.map((x) => File(x.path)).toList();
    final localPaths = picked.map((x) => x.path).toList();

    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      final resp = await _api.uploadImages(
          files: files, localPaths: localPaths, dedup: true);
      if (resp.summary.saved > 0) {
        PhotoSyncService.instance.notifyNewUploads(resp.summary.saved);
      }
      await _loadGalleryPhotos();
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _isLoading = false);
    }
  }

  // ================= SEARCH HANDLER (DEBOUNCE) =================

  void _onSearchChanged(String value) {
    if (_debounce?.isActive ?? false) _debounce!.cancel();
    _debounce = Timer(_debounceDuration, () {
      _searchByText(value);
    });
  }

  // ================= UI =================

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF121212),
      floatingActionButton: FloatingActionButton(
        heroTag: 'home_fab',
        backgroundColor: Colors.white,
        foregroundColor: Colors.black,
        onPressed: _uploadNewImages,
        child: const Icon(Icons.add_photo_alternate_outlined),
      ),
      body: SafeArea(
        child: Column(
          children: [
            const SizedBox(height: 8),
            _buildStatusIndicator(),
            const SizedBox(height: 8),
            _buildSearchRow(),
            const _SyncBanner(),
            const SizedBox(height: 16),
            Expanded(
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 250),
                switchInCurve: Curves.easeOut,
                switchOutCurve: Curves.easeIn,
                child: _buildContent(),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStatusIndicator() {
    Color statusColor;
    String statusText;
    String tooltip;

    if (!_isBackendOnline) {
      statusColor = Colors.red;
      statusText = 'Offline';
      tooltip = 'Backend is unreachable';
    } else if (_isWakingUp) {
      statusColor = Colors.orange;
      statusText = 'Waking up...';
      tooltip = 'ML services are performing a cold start (~2 mins)';
    } else {
      switch (_globalStatus) {
        case ImageProcessingStatus.unprocessed:
          statusColor = Colors.grey;
          statusText = 'Indexing...';
          tooltip = 'New images found. Search results may be incomplete.';
          break;
        case ImageProcessingStatus.fastEncoded:
          statusColor = Colors.yellow;
          statusText = 'Basic AI Search';
          tooltip = 'Deep AI contextualization in progress...';
          break;
        case ImageProcessingStatus.slowEncoded:
        case ImageProcessingStatus.inIndex:
          statusColor = Colors.green;
          statusText = 'Deep AI Search';
          tooltip = 'Gallery fully synchronized. All models active.';
          break;
        case ImageProcessingStatus.failed:
          statusColor = Colors.redAccent;
          statusText = 'Processing Error';
          tooltip = 'Some images failed to process.';
          break;
      }
    }

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          Tooltip(
            message: tooltip,
            preferBelow: false,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: statusColor.withOpacity(0.1),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: statusColor.withOpacity(0.3)),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(color: statusColor, shape: BoxShape.circle),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    statusText,
                    style: TextStyle(
                      color: statusColor,
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSearchRow() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Expanded(
            child: Container(
              height: 52,
              decoration: BoxDecoration(
                color: const Color(0xFF1E1E1E),
                borderRadius: BorderRadius.circular(18),
                border: Border.all(color: Colors.white10),
              ),
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Row(
                children: [
                  const Icon(Icons.search, color: Colors.white54),
                  const SizedBox(width: 8),
                  Expanded(
                    child: TextField(
                      controller: _searchController,
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 16,
                      ),
                      decoration: const InputDecoration(
                        hintText: 'Search photos...',
                        hintStyle: TextStyle(color: Colors.white38),
                        border: InputBorder.none,
                      ),
                      onSubmitted: (_) => _searchByText(_searchController.text),
                    ),
                  ),
                  if (_isSearchMode)
                    GestureDetector(
                      onTap: () {
                        _searchController.clear();
                        _loadGalleryPhotos();
                      },
                      child: const Icon(Icons.close, color: Colors.white54, size: 20),
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(width: 12),
          SizedBox(
            height: 52,
            width: 52,
            child: ClipRRect(
              borderRadius: BorderRadius.circular(18),
              child: Material(
                color: const Color(0xFF1E1E1E),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(18),
                  side: const BorderSide(color: Colors.white10),
                ),
                child: InkWell(
                  onTap: () => _searchByImage(),
                  child: const Center(
                    child: Icon(Icons.camera_alt_outlined, color: Colors.white),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildContent() {
    if (_isLoading) {
      return const Center(
        key: ValueKey('loading'),
        child: CircularProgressIndicator(color: Colors.white),
      );
    }

    if (_error != null) {
      return Center(
        key: const ValueKey('error'),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Text(
            _error!,
            textAlign: TextAlign.center,
            style: const TextStyle(color: Colors.redAccent),
          ),
        ),
      );
    }

    if (_isSearchMode) {
      return _buildSearchGrid();
    }

    return _buildGalleryGrid();
  }

  /// Grid showing device gallery photos.
  Widget _buildGalleryGrid() {
    if (_galleryAssets.isEmpty) {
      return const Center(
        key: ValueKey('empty-gallery'),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.photo_library_outlined, size: 48, color: Colors.white24),
            SizedBox(height: 16),
            Text('Gallery is empty',
                style: TextStyle(color: Colors.white54, fontSize: 16)),
          ],
        ),
      );
    }

    return Padding(
      key: const ValueKey('gallery-grid'),
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: GridView.builder(
        itemCount: _galleryAssets.length,
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 3,
          mainAxisSpacing: 2,
          crossAxisSpacing: 2,
        ),
        itemBuilder: (context, index) {
          final asset = _galleryAssets[index];
          return _GalleryThumbnail(
            asset: asset,
            isSynced: _syncedIds.contains(asset.id),
            onTap: () => _openAssetDetail(asset),
          );
        },
      ),
    );
  }

  /// Grid showing backend search results.
  Widget _buildSearchGrid() {
    if (_searchResults.isEmpty) {
      return const Center(
        key: ValueKey('empty-search'),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.search_off, size: 48, color: Colors.white24),
            SizedBox(height: 16),
            Text('No results found',
                style: TextStyle(color: Colors.white54, fontSize: 16)),
          ],
        ),
      );
    }

    return Padding(
      key: const ValueKey('search-grid'),
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: GridView.builder(
        itemCount: _searchResults.length,
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 3,
          mainAxisSpacing: 2,
          crossAxisSpacing: 2,
        ),
        itemBuilder: (context, index) {
          final item = _searchResults[index];
          return Hero(
            tag: item.url,
            child: Material(
              color: Colors.transparent,
              child: InkWell(
                onTap: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(
                      builder: (_) => DetailScreen(
                        imageUrl: item.url,
                        mediaItem: item,
                        onFindSimilar: (file) => _searchByImage(file),
                        onDelete: () => _searchByText(_searchController.text),
                      ),
                    ),
                  );
                },
                child: CachedImage(
                  localPath: item.localPath,
                  remoteUrl:
                      item.thumbUrl.isNotEmpty ? item.thumbUrl : item.url,
                  status: item.status,
                  fit: BoxFit.cover,
                  loadingBuilder: (context, child, progress) {
                    if (progress == null) return child;
                    return Container(color: const Color(0xFF1E1E1E));
                  },
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  /// Open a device gallery asset in DetailScreen.
  Future<void> _openAssetDetail(AssetEntity asset) async {
    final file = await asset.file;
    if (file == null || !mounted) return;

    final item = MediaItem(
      id: asset.id.hashCode,
      url: file.path,
      thumbUrl: file.path,
      mimeType: asset.mimeType ?? 'image/jpeg',
      sizeBytes: asset.size.width.toInt() * asset.size.height.toInt(),
      width: asset.size.width.toInt(),
      height: asset.size.height.toInt(),
      checksum: '',
      createdAt: asset.createDateTime.toIso8601String(),
      takenAt: asset.createDateTime.toIso8601String(),
      localPath: file.path,
    );

    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => DetailScreen(
          imageUrl: file.path,
          mediaItem: item,
          onFindSimilar: (f) => _searchByImage(f),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Sync banner – subscribes to PhotoSyncService progress stream
// ---------------------------------------------------------------------------

class _SyncBanner extends StatefulWidget {
  const _SyncBanner();

  @override
  State<_SyncBanner> createState() => _SyncBannerState();
}

class _SyncBannerState extends State<_SyncBanner> {
  late StreamSubscription<PhotoSyncProgress> _sub;
  PhotoSyncProgress _progress = PhotoSyncService.instance.lastProgress;
  Timer? _autoDismiss;

  @override
  void initState() {
    super.initState();
    _sub = PhotoSyncService.instance.progressStream.listen((p) {
      if (!mounted) return;
      setState(() => _progress = p);

      _autoDismiss?.cancel();
      if (p.status == PhotoSyncStatus.completed) {
        _autoDismiss = Timer(const Duration(seconds: 4), () {
          if (mounted) {
            setState(() =>
                _progress = const PhotoSyncProgress()); // back to idle
          }
        });
      } else if (p.status == PhotoSyncStatus.error) {
        _autoDismiss = Timer(const Duration(seconds: 5), () {
          if (mounted) {
            setState(() => _progress = const PhotoSyncProgress());
          }
        });
      }
    });
  }

  @override
  void dispose() {
    _sub.cancel();
    _autoDismiss?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_progress.status == PhotoSyncStatus.idle) {
      return const SizedBox.shrink();
    }

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: const Color(0xFF1E1E1E),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: Colors.white10),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                _buildLeadingIcon(),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    _progress.displayText,
                    style: const TextStyle(
                      color: Colors.white70,
                      fontSize: 13,
                    ),
                  ),
                ),
              ],
            ),
            if (_progress.status == PhotoSyncStatus.syncing) ...[
              const SizedBox(height: 8),
              ClipRRect(
                borderRadius: BorderRadius.circular(4),
                child: LinearProgressIndicator(
                  value: _progress.progressFraction,
                  backgroundColor: Colors.white12,
                  color: Colors.blueAccent,
                  minHeight: 4,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildLeadingIcon() {
    switch (_progress.status) {
      case PhotoSyncStatus.checking:
        return const SizedBox(
          width: 16,
          height: 16,
          child: CircularProgressIndicator(
            strokeWidth: 2,
            color: Colors.white54,
          ),
        );
      case PhotoSyncStatus.syncing:
        return const Icon(Icons.cloud_upload_outlined,
            size: 18, color: Colors.blueAccent);
      case PhotoSyncStatus.completed:
        return const Icon(Icons.cloud_done_outlined,
            size: 18, color: Colors.greenAccent);
      case PhotoSyncStatus.error:
        return const Icon(Icons.cloud_off_outlined,
            size: 18, color: Colors.redAccent);
      case PhotoSyncStatus.idle:
        return const SizedBox.shrink();
    }
  }
}

/// Thumbnail widget that loads from a device gallery [AssetEntity].
class _GalleryThumbnail extends StatefulWidget {
  final AssetEntity asset;
  final bool isSynced;
  final VoidCallback onTap;

  const _GalleryThumbnail({
    required this.asset,
    required this.isSynced,
    required this.onTap,
  });

  @override
  State<_GalleryThumbnail> createState() => _GalleryThumbnailState();
}

class _GalleryThumbnailState extends State<_GalleryThumbnail> {
  Uint8List? _thumbBytes;

  @override
  void initState() {
    super.initState();
    _loadThumb();
  }

  Future<void> _loadThumb() async {
    final bytes = await widget.asset.thumbnailDataWithSize(
      const ThumbnailSize(300, 300),
    );
    if (mounted && bytes != null) {
      setState(() => _thumbBytes = bytes);
    }
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: widget.onTap,
      child: Stack(
        fit: StackFit.expand,
        children: [
          _thumbBytes != null
              ? Image.memory(_thumbBytes!, fit: BoxFit.cover)
              : Container(color: const Color(0xFF1E1E1E)),
          if (widget.isSynced)
            Positioned(
              bottom: 4,
              right: 4,
              child: Container(
                padding: const EdgeInsets.all(2),
                decoration: const BoxDecoration(
                  color: Colors.black54,
                  shape: BoxShape.circle,
                ),
                child: const Icon(
                  Icons.cloud_done,
                  size: 12,
                  color: Colors.greenAccent,
                ),
              ),
            ),
        ],
      ),
    );
  }
}
