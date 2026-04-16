import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:frontend_flutter/core/category_cache_service.dart';
import 'package:frontend_flutter/core/photo_sync.dart';
import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/data/models.dart';
import 'package:frontend_flutter/ui/screens/category_photos_screen.dart';
import 'package:frontend_flutter/ui/widgets/cached_image.dart';

class CategoriesScreen extends StatefulWidget {
  const CategoriesScreen({super.key});

  @override
  State<CategoriesScreen> createState() => _CategoriesScreenState();
}

class _CategoriesScreenState extends State<CategoriesScreen> {
  final ApiClient _api = ApiClient();
  bool _isLoading = true;
  bool _isRefreshing = false;
  String? _error;
  List<PhotoCategory> _categories = [];
  Map<String, String> _renames = {};
  List<Map<String, dynamic>> _customQueries = [];
  StreamSubscription<PhotoSyncProgress>? _syncSub;

  static const _prefsRenamesKey = 'categories_renames';
  static const _prefsCustomKey = 'categories_custom';

  /// Query-to-icon map for resolving icons in the UI layer.
  static final Map<String, IconData> _queryToIcon = {
    'person portrait': Icons.people,
    'selfie': Icons.face,
    'dog': Icons.pets,
    'cat kitten': Icons.pets,
    'food meal dish': Icons.restaurant,
    'car vehicle': Icons.directions_car,
    'landscape nature': Icons.landscape,
    'beach ocean': Icons.beach_access,
    'sunset sunrise': Icons.wb_twilight,
    'mountain hiking': Icons.terrain,
    'flower garden': Icons.local_florist,
    'building architecture': Icons.location_city,
    'city street urban': Icons.apartment,
    'screenshot': Icons.screenshot,
    'document text paper': Icons.description,
    'sport athlete': Icons.sports,
    'night dark': Icons.nightlight,
    'art painting drawing': Icons.palette,
    'snow winter': Icons.ac_unit,
    'water river lake': Icons.water,
  };

  @override
  void initState() {
    super.initState();
    _initFromCache();
    _listenToSync();
  }

  @override
  void dispose() {
    _syncSub?.cancel();
    _api.close();
    super.dispose();
  }

  /// Listen to PhotoSyncService. When sync completes with new uploads,
  /// wait 30s for embeddings, then background refresh.
  void _listenToSync() {
    _syncSub = PhotoSyncService.instance.progressStream.listen((progress) {
      if (progress.status == PhotoSyncStatus.completed &&
          progress.syncedPhotos > 0) {
        Future.delayed(const Duration(seconds: 30), () {
          if (mounted) _backgroundRefresh();
        });
      }
    });
  }

  // -----------------------------------------------------------------------
  // Cache-first loading
  // -----------------------------------------------------------------------

  Future<void> _initFromCache() async {
    await _loadPrefs();

    // Try to load from cache first
    final cached = await CategoryCacheService.loadCache();
    if (cached != null && cached.isNotEmpty) {
      final displayed = _convertCachedToDisplay(cached);
      if (mounted) {
        setState(() {
          _categories = displayed;
          _isLoading = false;
        });
      }

      // Intentionally skip background refresh on init — user can pull-to-refresh.
    } else {
      // No cache — show empty state; user can pull-to-refresh manually.
      if (mounted) setState(() => _isLoading = false);
    }
  }

  /// Background refresh: subtle spinner, no full-screen loading state.
  Future<void> _backgroundRefresh() async {
    if (_isRefreshing) return;
    if (mounted) setState(() => _isRefreshing = true);

    try {
      final result = await CategoryCacheService.refreshCategories(
        customQueries: _customQueries,
      );
      final displayed = _convertCachedToDisplay(result.categories);
      if (mounted) {
        setState(() {
          _categories = displayed;
          _isRefreshing = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _isRefreshing = false);
    }
  }

  /// Full refresh: shows loading indicator, used on first load and pull-to-refresh.
  Future<void> _fullRefresh() async {
    setState(() {
      _isLoading = true;
      _error = null;
    });

    try {
      await _loadPrefs();
      final result = await CategoryCacheService.refreshCategories(
        customQueries: _customQueries,
      );
      final displayed = _convertCachedToDisplay(result.categories);
      if (mounted) {
        setState(() {
          _categories = displayed;
          _isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _isLoading = false;
        });
      }
    }
  }

  // -----------------------------------------------------------------------
  // Helpers
  // -----------------------------------------------------------------------

  /// Convert CachedCategory list → PhotoCategory list with icon resolution,
  /// rename application, and filtering (hide empty built-ins).
  List<PhotoCategory> _convertCachedToDisplay(List<CachedCategory> cached) {
    final builtInQueries =
        CategoryCacheService.builtInConfigs.map((c) => c.query).toSet();
    final filtered = <PhotoCategory>[];

    for (final cat in cached) {
      final isBuiltIn = builtInQueries.contains(cat.query);
      // Hide empty built-in categories
      if (isBuiltIn && cat.count == 0) continue;

      final icon = _queryToIcon[cat.query] ?? Icons.label;
      final rename = _renames[cat.query];
      final displayName = rename ?? cat.name;

      filtered.add(PhotoCategory(
        name: displayName,
        icon: icon,
        coverUrl: cat.coverUrl,
        count: cat.count,
        query: cat.query,
        results: cat.results,
      ));
    }
    return filtered;
  }

  Future<void> _loadPrefs() async {
    final prefs = await SharedPreferences.getInstance();

    final renamesJson = prefs.getString(_prefsRenamesKey);
    if (renamesJson != null) {
      final decoded = jsonDecode(renamesJson) as Map<String, dynamic>;
      _renames = decoded.map((k, v) => MapEntry(k, v as String));
    } else {
      _renames = {};
    }

    final customJson = prefs.getString(_prefsCustomKey);
    if (customJson != null) {
      _customQueries =
          (jsonDecode(customJson) as List<dynamic>)
              .cast<Map<String, dynamic>>();
    } else {
      _customQueries = [];
    }
  }

  Future<void> _saveRenames() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefsRenamesKey, jsonEncode(_renames));
  }

  Future<void> _saveCustomQueries() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefsCustomKey, jsonEncode(_customQueries));
  }

  bool _isCustomCategory(PhotoCategory cat) {
    return _customQueries.any((c) => c['query'] == cat.query);
  }

  Future<void> _addCustomCategory() async {
    final controller = TextEditingController();
    final name = await showDialog<String>(
      context: context,
      builder:
          (ctx) => AlertDialog(
            backgroundColor: const Color(0xFF1E1E1E),
            title: const Text(
              'New Category',
              style: TextStyle(color: Colors.white),
            ),
            content: TextField(
              controller: controller,
              autofocus: true,
              style: const TextStyle(color: Colors.white),
              decoration: const InputDecoration(
                hintText: 'e.g. birthday party',
                hintStyle: TextStyle(color: Colors.white38),
                enabledBorder: UnderlineInputBorder(
                  borderSide: BorderSide(color: Colors.white24),
                ),
                focusedBorder: UnderlineInputBorder(
                  borderSide: BorderSide(color: Colors.white),
                ),
              ),
              onSubmitted: (v) => Navigator.pop(ctx, v.trim()),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text(
                  'Cancel',
                  style: TextStyle(color: Colors.white54),
                ),
              ),
              TextButton(
                onPressed: () => Navigator.pop(ctx, controller.text.trim()),
                child: const Text('Add'),
              ),
            ],
          ),
    );

    if (name == null || name.isEmpty) return;

    // Check for duplicate query
    final alreadyExists =
        _categories.any((c) => c.query == name) ||
        _customQueries.any((c) => c['query'] == name);
    if (alreadyExists) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Category "$name" already exists')),
        );
      }
      return;
    }

    // Save to prefs
    _customQueries.add({'name': name, 'query': name});
    await _saveCustomQueries();

    // Run search and add to list (with relative scoring via service refresh)
    // For a single category, do a quick inline search
    try {
      final resp = await _api.textSearch(name, limit: 50, threshold: 0);
      // Absolute floor + relative scoring (same logic as CategoryCacheService)
      var relevant = resp.results
          .where((r) => (r.score ?? 0) >= CategoryCacheService.minAbsoluteScore)
          .toList();
      if (relevant.length >= 2) {
        final scores = relevant.map((r) => r.score ?? 0.0).toList();
        final mean = scores.reduce((a, b) => a + b) / scores.length;
        final variance =
            scores.map((s) => (s - mean) * (s - mean)).reduce((a, b) => a + b) /
                scores.length;
        final stddev = sqrt(variance);
        final cutoff = mean + 1.5 * stddev;
        relevant = relevant.where((r) => (r.score ?? 0) >= cutoff).toList();
      }
      final coverUrl =
          relevant.isNotEmpty
              ? (relevant.first.thumbUrl.isNotEmpty
                  ? relevant.first.thumbUrl
                  : relevant.first.url)
              : '';
      if (mounted) {
        setState(() {
          _categories.add(
            PhotoCategory(
              name: name,
              icon: Icons.label,
              coverUrl: coverUrl,
              count: relevant.length,
              query: name,
              results: relevant,
            ),
          );
        });
      }
    } catch (_) {
      if (mounted) {
        setState(() {
          _categories.add(
            PhotoCategory(
              name: name,
              icon: Icons.label,
              coverUrl: '',
              count: 0,
              query: name,
            ),
          );
        });
      }
    }
  }

  void _showCategoryOptions(PhotoCategory category) {
    final isCustom = _isCustomCategory(category);

    showModalBottomSheet(
      context: context,
      backgroundColor: const Color(0xFF1E1E1E),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder:
          (ctx) => SafeArea(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 40,
                  height: 4,
                  margin: const EdgeInsets.only(top: 12, bottom: 8),
                  decoration: BoxDecoration(
                    color: Colors.white24,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 8,
                  ),
                  child: Text(
                    category.name,
                    style: const TextStyle(
                      color: Colors.white,
                      fontSize: 18,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                ListTile(
                  leading: const Icon(Icons.edit, color: Colors.white70),
                  title: const Text(
                    'Rename',
                    style: TextStyle(color: Colors.white),
                  ),
                  onTap: () {
                    Navigator.pop(ctx);
                    _renameCategory(category);
                  },
                ),
                if (isCustom)
                  ListTile(
                    leading: const Icon(Icons.delete, color: Colors.redAccent),
                    title: const Text(
                      'Delete',
                      style: TextStyle(color: Colors.redAccent),
                    ),
                    onTap: () {
                      Navigator.pop(ctx);
                      _deleteCustomCategory(category);
                    },
                  ),
                const SizedBox(height: 8),
              ],
            ),
          ),
    );
  }

  Future<void> _renameCategory(PhotoCategory category) async {
    final controller = TextEditingController(text: category.name);
    final newName = await showDialog<String>(
      context: context,
      builder:
          (ctx) => AlertDialog(
            backgroundColor: const Color(0xFF1E1E1E),
            title: const Text(
              'Rename Category',
              style: TextStyle(color: Colors.white),
            ),
            content: TextField(
              controller: controller,
              autofocus: true,
              style: const TextStyle(color: Colors.white),
              decoration: const InputDecoration(
                enabledBorder: UnderlineInputBorder(
                  borderSide: BorderSide(color: Colors.white24),
                ),
                focusedBorder: UnderlineInputBorder(
                  borderSide: BorderSide(color: Colors.white),
                ),
              ),
              onSubmitted: (v) => Navigator.pop(ctx, v.trim()),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text(
                  'Cancel',
                  style: TextStyle(color: Colors.white54),
                ),
              ),
              TextButton(
                onPressed: () => Navigator.pop(ctx, controller.text.trim()),
                child: const Text('Save'),
              ),
            ],
          ),
    );

    if (newName == null || newName.isEmpty || newName == category.name) return;

    // Persist rename
    _renames[category.query] = newName;
    await _saveRenames();

    // Update in-memory
    if (mounted) {
      setState(() {
        final idx = _categories.indexWhere((c) => c.query == category.query);
        if (idx != -1) {
          final old = _categories[idx];
          _categories[idx] = PhotoCategory(
            name: newName,
            icon: old.icon,
            coverUrl: old.coverUrl,
            count: old.count,
            query: old.query,
            results: old.results,
          );
        }
      });
    }
  }

  Future<void> _deleteCustomCategory(PhotoCategory category) async {
    _customQueries.removeWhere((c) => c['query'] == category.query);
    _renames.remove(category.query);
    await _saveCustomQueries();
    await _saveRenames();

    if (mounted) {
      setState(() {
        _categories.removeWhere((c) => c.query == category.query);
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF121212),
      appBar: AppBar(
        title: const Text('Categories'),
        actions: [
          if (_isRefreshing)
            const Padding(
              padding: EdgeInsets.only(right: 16),
              child: SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  color: Colors.white54,
                ),
              ),
            ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        heroTag: 'categories_fab',
        onPressed: _addCustomCategory,
        backgroundColor: Colors.white12,
        child: const Icon(Icons.add, color: Colors.white),
      ),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_isLoading) {
      return const Center(
        child: CircularProgressIndicator(color: Colors.white),
      );
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.cloud_off, color: Colors.white24, size: 48),
            const SizedBox(height: 16),
            Text(_error!, style: const TextStyle(color: Colors.redAccent)),
            const SizedBox(height: 16),
            OutlinedButton(
              onPressed: _fullRefresh,
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.white,
                side: const BorderSide(color: Colors.white24),
              ),
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    if (_categories.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.category, color: Colors.white24, size: 48),
            const SizedBox(height: 16),
            const Text(
              'No categories discovered yet.\nUpload photos to see them here.',
              textAlign: TextAlign.center,
              style: TextStyle(color: Colors.white54, fontSize: 16),
            ),
            const SizedBox(height: 16),
            OutlinedButton(
              onPressed: _fullRefresh,
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.white,
                side: const BorderSide(color: Colors.white24),
              ),
              child: const Text('Refresh'),
            ),
          ],
        ),
      );
    }

    return RefreshIndicator(
      onRefresh: _fullRefresh,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: GridView.builder(
          itemCount: _categories.length,
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 2,
            mainAxisSpacing: 12,
            crossAxisSpacing: 12,
            childAspectRatio: 1.0,
          ),
          itemBuilder: (context, index) {
            final cat = _categories[index];
            return _CategoryTile(
              category: cat,
              onLongPress: () => _showCategoryOptions(cat),
            );
          },
        ),
      ),
    );
  }
}

class _CategoryTile extends StatelessWidget {
  final PhotoCategory category;
  final VoidCallback onLongPress;

  const _CategoryTile({required this.category, required this.onLongPress});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: () {
            Navigator.push(
              context,
              MaterialPageRoute(
                builder: (_) => CategoryPhotosScreen(category: category),
              ),
            );
          },
          onLongPress: onLongPress,
          child: Stack(
            fit: StackFit.expand,
            children: [
              if (category.coverUrl.isNotEmpty)
                CachedImage(
                  remoteUrl: category.coverUrl,
                  fit: BoxFit.cover,
                )
              else
                Container(
                  color: const Color(0xFF2A2A2A),
                  child: Center(
                    child: Icon(
                      category.icon as IconData,
                      color: Colors.white24,
                      size: 48,
                    ),
                  ),
                ),
              // Gradient overlay
              Container(
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Colors.transparent,
                      Colors.black.withValues(alpha: 0.7),
                    ],
                  ),
                ),
              ),
              // Icon + name + count
              Positioned(
                left: 12,
                right: 12,
                bottom: 12,
                child: Row(
                  children: [
                    Icon(
                      category.icon as IconData,
                      color: Colors.white,
                      size: 20,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        category.name,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 16,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    Text(
                      '${category.count}',
                      style: const TextStyle(
                        color: Colors.white70,
                        fontSize: 14,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
