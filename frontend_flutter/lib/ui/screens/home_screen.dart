import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/screens/detail_screen.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';

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
  String? _error;

  /// URLs that we show in the grid (either full list or search results).
  List<MediaItem> _mediaItems = [];

  @override
  void initState() {
    super.initState();
    _checkBackendStatus();
    _loadAllImages();
  }

  Future<void> _checkBackendStatus() async {
    final isOnline = await _api.checkHealth();
    if (mounted) {
      setState(() {
        _isBackendOnline = isOnline;
      });
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    _debounce?.cancel();
    _api.close();
    super.dispose();
  }

  // ================= DATA LOADING =================

  /// Home gallery: all user images from backend.
  Future<void> _loadAllImages() async {
    setState(() {
      _isLoading = true;
      _error = null;
      _isSearchMode = false;
    });

    try {
      final List<MediaItem> items = await ApiClient.listUserImages(
        userId: AppConfig.userId,
        limit: 100,
        offset: 0,
      );

      setState(() {
        _mediaItems = items;
        _isBackendOnline = true; // If this succeeds, we are definitely online
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
        // Don't set offline here immediately, as it could be just this call
      });
      _checkBackendStatus(); // Re-check health specifically
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  /// Text search using /api/search/text
  Future<void> _searchByText(String query) async {
    final trimmed = query.trim();
    if (trimmed.isEmpty) {
      _loadAllImages();
      return;
    }
    
    // If query is too short (backend requires min 2 chars), show blank screen
    if (trimmed.length < 2) {
      setState(() {
        _mediaItems = [];
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
      final SearchResponse resp = await _api.textSearch(trimmed);
      final filtered = _filterResults(resp.results);

      setState(() {
        _mediaItems = filtered.map((r) => MediaItem(
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
        )).toList();
        _isBackendOnline = true;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  /// Image search using /api/search/image
  Future<void> _searchByImage([File? preSelectedFile]) async {
    File? file;
    
    if (preSelectedFile != null) {
      file = preSelectedFile;
    } else {
      final XFile? picked = await _picker.pickImage(source: ImageSource.gallery);
      if (picked == null) return;
      file = File(picked.path);
    }

    setState(() {
      _isLoading = true;
      _error = null;
      _isSearchMode = true;
      // _searchController.text = "Searching by image..."; // Removed as per request
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
      // Use a slightly more lenient threshold for image search (0.15) vs text search (0.12)
      final filtered = _filterResults(resp.results, threshold: 0.25);

      setState(() {
        _mediaItems = filtered.map((r) => MediaItem(
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
        )).toList();
        _isBackendOnline = true;
      });
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
    }
  }

  /// Progressive Drop Filter
  /// Cuts off results when the score drops by more than [threshold] compared to the previous item.
  List<MediaResponse> _filterResults(List<MediaResponse> results, {double threshold = 0.12}) {
    if (results.isEmpty) return [];

    // 1. Sort by score descending (just in case backend didn't)
    results.sort((a, b) => (b.score ?? 0).compareTo(a.score ?? 0));

    List<MediaResponse> kept = [];
    
    // Always keep the first (best) result
    kept.add(results[0]);

    for (int i = 1; i < results.length; i++) {
      final prevScore = results[i - 1].score ?? 0;
      final currScore = results[i].score ?? 0;
      final drop = prevScore - currScore;

      if (drop > threshold) {
        // Significant drop detected! Stop adding results.
        break;
      }
      kept.add(results[i]);
    }

    return kept;
  }

  /// Upload button (bottom bar) – upload new images to cloud, then refresh.
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
      await _api.uploadImages(files: files, localPaths: localPaths, dedup: true);
      await _loadAllImages();
    } catch (e) {
      setState(() {
        _error = e.toString();
      });
    } finally {
      setState(() {
        _isLoading = false;
      });
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
    final bottomPadding = MediaQuery.of(context).padding.bottom;

    return Scaffold(
      backgroundColor: const Color(0xFF121212),
      body: SafeArea(
        child: Column(
          children: [
            const SizedBox(height: 8),
            _buildStatusIndicator(),
            const SizedBox(height: 8),
            _buildSearchRow(),
            const SizedBox(height: 16),
            Expanded(
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 250),
                switchInCurve: Curves.easeOut,
                switchOutCurve: Curves.easeIn,
                child: _buildContent(),
              ),
            ),
            _buildBottomBar(bottomPadding),
          ],
        ),
      ),
    );
  }

  Widget _buildStatusIndicator() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: _isBackendOnline ? Colors.green.withOpacity(0.1) : Colors.red.withOpacity(0.1),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(
                color: _isBackendOnline ? Colors.green.withOpacity(0.3) : Colors.red.withOpacity(0.3),
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 8,
                  height: 8,
                  decoration: BoxDecoration(
                    color: _isBackendOnline ? Colors.green : Colors.red,
                    shape: BoxShape.circle,
                  ),
                ),
                const SizedBox(width: 6),
                Text(
                  _isBackendOnline ? 'Online' : 'Offline',
                  style: TextStyle(
                    color: _isBackendOnline ? Colors.green : Colors.red,
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ],
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
          // SearchBar
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
                ],
              ),
            ),
          ),
          const SizedBox(width: 12),
          // Image Search Icon
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
                    child: Icon(
                      Icons.camera_alt_outlined,
                      color: Colors.white,
                    ),
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

    if (_mediaItems.isEmpty) {
      return Center(
        key: const ValueKey('empty'),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              _isSearchMode ? Icons.search_off : Icons.photo_library_outlined,
              size: 48,
              color: Colors.white24,
            ),
            const SizedBox(height: 16),
            Text(
              _isSearchMode ? 'No results found' : 'Gallery is empty',
              style: const TextStyle(
                color: Colors.white54,
                fontSize: 16,
              ),
            ),
          ],
        ),
      );
    }

    return Padding(
      key: const ValueKey('grid'),
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: GridView.builder(
        itemCount: _mediaItems.length,
        gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: 3,
          mainAxisSpacing: 2,
          crossAxisSpacing: 2,
          childAspectRatio: 1.0, // Square tiles for cleaner look
        ),
        itemBuilder: (context, index) {
          final item = _mediaItems[index];
          // Fix URL for Android Emulator
          final url = AppConfig.fixImageUrl(
            item.thumbUrl.isNotEmpty ? item.thumbUrl : item.url,
          );

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
                      ),
                    ),
                  );
                },
                child: CachedImage(
                  localPath: item.localPath,
                  remoteUrl: item.thumbUrl.isNotEmpty ? item.thumbUrl : item.url,
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

  Widget _buildBottomBar(double bottomPadding) {
    return Container(
      padding: EdgeInsets.fromLTRB(16, 12, 16, 12 + bottomPadding),
      decoration: const BoxDecoration(
        color: Color(0xFF1E1E1E),
        border: Border(top: BorderSide(color: Colors.white10)),
      ),
      child: Row(
        children: [
          Expanded(
            child: _BottomBarButton(
              icon: Icons.grid_view,
              label: 'Gallery',
              isActive: !_isSearchMode,
              onTap: () {
                _searchController.clear();
                _loadAllImages();
              },
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _BottomBarButton(
              icon: Icons.add_photo_alternate_outlined,
              label: 'Upload',
              isActive: false,
              onTap: _uploadNewImages,
            ),
          ),
        ],
      ),
    );
  }
}

class _BottomBarButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final bool isActive;
  final VoidCallback onTap;

  const _BottomBarButton({
    required this.icon,
    required this.label,
    required this.isActive,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        height: 48,
        decoration: BoxDecoration(
          color: isActive ? Colors.white : Colors.transparent,
          borderRadius: BorderRadius.circular(12),
          border: isActive ? null : Border.all(color: Colors.white24),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              icon, 
              color: isActive ? Colors.black : Colors.white,
              size: 20,
            ),
            const SizedBox(width: 8),
            Text(
              label,
              style: TextStyle(
                color: isActive ? Colors.black : Colors.white,
                fontWeight: FontWeight.w600,
                fontSize: 14,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
