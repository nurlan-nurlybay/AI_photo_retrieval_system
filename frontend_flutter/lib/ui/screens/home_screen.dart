import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:photo_manager/photo_manager.dart';

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

  /// Backend search results (shown in search mode).
  List<MediaItem> _searchResults = [];

  @override
  void initState() {
    super.initState();
    _checkBackendStatus();
    _loadGalleryPhotos();
    _triggerAutoSync();
    _initStatusStream();
  }

  void _initStatusStream() {
    StatusStreamService.instance.start();
    _statusSub = StatusStreamService.instance.stream.listen((update) {
      if (!mounted) return;
      setState(() {
        // Update specific item in search results if it exists
        final index = _searchResults.indexWhere((item) => item.id == update.mediaId);
        if (index != -1) {
          _searchResults[index].status = update.status;
        }
        _updateGlobalStatus();
      });
    });
  }

  void _updateGlobalStatus() {
    if (_searchResults.isEmpty) {
      _globalStatus = ImageProcessingStatus.inIndex;
      return;
    }
    // Logic: lowest common denominator
    // If ANY is unprocessed -> Grey
    // Else if ANY is fastEncoded -> Yellow
    // Else -> Green
    if (_searchResults.any((item) => item.status == ImageProcessingStatus.unprocessed)) {
      _globalStatus = ImageProcessingStatus.unprocessed;
    } else if (_searchResults.any((item) => item.status == ImageProcessingStatus.fastEncoded)) {
      _globalStatus = ImageProcessingStatus.fastEncoded;
    } else {
      _globalStatus = ImageProcessingStatus.inIndex;
    }
  }

  Future<void> _triggerAutoSync() async {
    final isOnline = await _api.checkHealth();
    if (isOnline) PhotoSyncService.instance.startSync();
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
    if (trimmed.isEmpty) {
      _loadGalleryPhotos();
      return;
    }

    if (trimmed.length < 2) {
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
      final SearchResponse resp = await _api.textSearch(trimmed, limit: 30);
      final filtered = _filterResults(resp.results);

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
                  status: r.status,
                ))
            .toList();
        _isBackendOnline = true;
        _updateGlobalStatus();
      });
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _isLoading = false);
    }
  }

  /// Image search using /api/search/image
  Future<void> _searchByImage([File? preSelectedFile]) async {
    File? file;

    if (preSelectedFile != null) {
      file = preSelectedFile;
    } else {
      final XFile? picked =
          await _picker.pickImage(source: ImageSource.gallery);
      if (picked == null) return;
      file = File(picked.path);
    }

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
      final SearchResponse resp = await _api.imageSearch(file);
      final filtered = _filterResults(resp.results, threshold: 0.25);

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
                  status: r.status,
                ))
            .toList();
        _isBackendOnline = true;
        _updateGlobalStatus();
      });
    } catch (e) {
      setState(() => _error = e.toString());
    } finally {
      setState(() => _isLoading = false);
    }
  }

  /// Minimum cosine-similarity to be considered a real match.
  static const double _minScore = 0.22;

  List<MediaResponse> _filterResults(List<MediaResponse> results,
      {double threshold = 0.12}) {
    if (results.isEmpty) return [];
    results.sort((a, b) => (b.score ?? 0).compareTo(a.score ?? 0));

    // 1. Absolute floor — drop anything below minimum score
    // Updated: If score is null, we assume the backend already approved it as relevant.
    final aboveFloor = results
        .where((r) => r.score == null || r.score! >= _minScore)
        .toList();
    if (aboveFloor.isEmpty) return [];

    // 2. Drop-based cutoff — stop when there's a big gap between consecutive scores
    // Only apply cutoff if scores are actually being provided
    List<MediaResponse> kept = [aboveFloor[0]];
    for (int i = 1; i < aboveFloor.length; i++) {
      if (aboveFloor[i-1].score == null || aboveFloor[i].score == null) {
        kept.add(aboveFloor[i]);
        continue;
      }
      final drop = aboveFloor[i - 1].score! - aboveFloor[i].score!;
      if (drop > threshold) break;
      kept.add(aboveFloor[i]);
    }
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
                      onChanged: _onSearchChanged,
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
  final VoidCallback onTap;

  const _GalleryThumbnail({required this.asset, required this.onTap});

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
      child: _thumbBytes != null
          ? Image.memory(_thumbBytes!, fit: BoxFit.cover)
          : Container(color: const Color(0xFF1E1E1E)),
    );
  }
}
