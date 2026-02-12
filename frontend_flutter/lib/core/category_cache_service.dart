import 'dart:convert';
import 'dart:math';

import 'package:shared_preferences/shared_preferences.dart';

import 'package:frontend_flutter/core/photo_sync.dart';
import 'package:frontend_flutter/data/api_client.dart';
import 'package:frontend_flutter/data/models.dart';

// ---------------------------------------------------------------------------
// Cache data model
// ---------------------------------------------------------------------------

class CachedCategory {
  final String name;
  final String query;
  final String coverUrl;
  final int count;
  final List<int> photoIds;
  final List<MediaResponse> results;

  CachedCategory({
    required this.name,
    required this.query,
    required this.coverUrl,
    required this.count,
    required this.photoIds,
    required this.results,
  });

  factory CachedCategory.fromJson(Map<String, dynamic> json) {
    final resultsList = (json['results'] as List<dynamic>? ?? [])
        .map((e) => MediaResponse.fromJson(e as Map<String, dynamic>))
        .toList();
    return CachedCategory(
      name: json['name'] as String,
      query: json['query'] as String,
      coverUrl: json['cover_url'] as String? ?? '',
      count: json['count'] as int? ?? 0,
      photoIds:
          (json['photo_ids'] as List<dynamic>? ?? []).cast<int>(),
      results: resultsList,
    );
  }

  Map<String, dynamic> toJson() => {
        'name': name,
        'query': query,
        'cover_url': coverUrl,
        'count': count,
        'photo_ids': photoIds,
        'results': results.map((r) => r.toJson()).toList(),
      };
}

class CategoryRefreshResult {
  final List<CachedCategory> categories;
  final Map<String, int> newPhotoCounts;

  CategoryRefreshResult({
    required this.categories,
    required this.newPhotoCounts,
  });
}

// ---------------------------------------------------------------------------
// Built-in category configs (no IconData — safe for isolates)
// ---------------------------------------------------------------------------

class CategoryConfig {
  final String name;
  final String query;

  const CategoryConfig(this.name, this.query);
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

class CategoryCacheService {
  static const _cacheKey = 'categories_cache_json';
  static const _cacheSyncedCountKey = 'categories_cache_synced_count';
  static const _prefSyncedIds = 'photo_sync_synced_ids';

  static const List<CategoryConfig> builtInConfigs = [
    CategoryConfig('People', 'person portrait'),
    CategoryConfig('Selfies', 'selfie'),
    CategoryConfig('Dogs', 'dog'),
    CategoryConfig('Cats', 'cat kitten'),
    CategoryConfig('Food', 'food meal dish'),
    CategoryConfig('Cars', 'car vehicle'),
    CategoryConfig('Nature', 'landscape nature'),
    CategoryConfig('Beach', 'beach ocean'),
    CategoryConfig('Sunset', 'sunset sunrise'),
    CategoryConfig('Mountains', 'mountain hiking'),
    CategoryConfig('Flowers', 'flower garden'),
    CategoryConfig('Buildings', 'building architecture'),
    CategoryConfig('City', 'city street urban'),
    CategoryConfig('Screenshots', 'screenshot'),
    CategoryConfig('Documents', 'document text paper'),
    CategoryConfig('Sports', 'sport athlete'),
    CategoryConfig('Night', 'night dark'),
    CategoryConfig('Art', 'art painting drawing'),
    CategoryConfig('Snow', 'snow winter'),
    CategoryConfig('Water', 'water river lake'),
  ];

  /// Load cached category results from SharedPreferences.
  static Future<List<CachedCategory>?> loadCache() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_cacheKey);
    if (raw == null) return null;
    try {
      final list = (jsonDecode(raw) as List<dynamic>)
          .map((e) => CachedCategory.fromJson(e as Map<String, dynamic>))
          .toList();
      return list;
    } catch (_) {
      return null;
    }
  }

  /// Persist category results and record synced photo count.
  static Future<void> saveCache(List<CachedCategory> categories) async {
    final prefs = await SharedPreferences.getInstance();
    final json = jsonEncode(categories.map((c) => c.toJson()).toList());
    await prefs.setString(_cacheKey, json);

    final syncedIds = prefs.getStringList(_prefSyncedIds) ?? [];
    await prefs.setInt(_cacheSyncedCountKey, syncedIds.length);

    // Clear the refresh flag — categories are now up-to-date
    await prefs.setBool(PhotoSyncService.prefNeedsCategoryRefresh, false);
  }

  /// Compare current synced photo count with count at time of last cache,
  /// and check the persistent "needs refresh" flag set by any upload path.
  static Future<bool> isRefreshNeeded() async {
    final prefs = await SharedPreferences.getInstance();

    // Check flag set by manual or auto-sync uploads
    if (prefs.getBool(PhotoSyncService.prefNeedsCategoryRefresh) == true) {
      return true;
    }

    final cachedCount = prefs.getInt(_cacheSyncedCountKey) ?? -1;
    final syncedIds = prefs.getStringList(_prefSyncedIds) ?? [];

    // No cache yet → need refresh
    if (cachedCount < 0) return true;
    // New photos synced since last cache
    return syncedIds.length != cachedCount;
  }

  /// Number of new photos synced since last category cache.
  /// Returns 0 if no cache exists yet (first run).
  static Future<int> newPhotoCount() async {
    final prefs = await SharedPreferences.getInstance();
    final cachedCount = prefs.getInt(_cacheSyncedCountKey) ?? 0;
    final syncedIds = prefs.getStringList(_prefSyncedIds) ?? [];
    final diff = syncedIds.length - cachedCount;
    return diff > 0 ? diff : 0;
  }

  /// Minimum cosine-similarity score for a result to be considered relevant.
  /// CLIP text-image scores below this are noise regardless of relative ranking.
  static const double minAbsoluteScore = 0.20;

  /// Run batched text searches with relative scoring.
  /// [customQueries] are user-created categories from SharedPreferences.
  /// Returns a [CategoryRefreshResult] with categories and diff counts.
  static Future<CategoryRefreshResult> refreshCategories({
    List<Map<String, dynamic>> customQueries = const [],
  }) async {
    final api = ApiClient();

    try {
      // Load old cache for diff comparison
      final oldCache = await loadCache();
      final oldPhotoIdsByQuery = <String, Set<int>>{};
      if (oldCache != null) {
        for (final c in oldCache) {
          oldPhotoIdsByQuery[c.query] = c.photoIds.toSet();
        }
      }

      // Merge built-in + custom into configs
      final allConfigs = <Map<String, String>>[
        for (final c in builtInConfigs)
          {'name': c.name, 'query': c.query, '_isCustom': 'false'},
        for (final c in customQueries)
          {
            'name': c['name'] as String,
            'query': c['query'] as String,
            '_isCustom': 'true',
          },
      ];

      // Process in batches of 4
      const batchSize = 4;
      final results = <CachedCategory>[];

      for (var i = 0; i < allConfigs.length; i += batchSize) {
        final batch = allConfigs.sublist(
          i,
          i + batchSize > allConfigs.length ? allConfigs.length : i + batchSize,
        );
        final batchResults = await Future.wait(batch.map((c) async {
          try {
            final resp = await api.textSearch(
              c['query']!,
              limit: 50,
              threshold: 0,
            );
            if (resp.results.isEmpty) {
              return CachedCategory(
                name: c['name']!,
                query: c['query']!,
                coverUrl: '',
                count: 0,
                photoIds: [],
                results: [],
              );
            }

            // Filter: absolute minimum + relative scoring (mean + 1.5*stddev)
            final aboveFloor = resp.results
                .where((r) => (r.score ?? 0) >= minAbsoluteScore)
                .toList();

            List<MediaResponse> relevant;
            if (aboveFloor.length >= 2) {
              final scores =
                  aboveFloor.map((r) => r.score ?? 0.0).toList();
              final mean = scores.reduce((a, b) => a + b) / scores.length;
              final variance = scores
                      .map((s) => (s - mean) * (s - mean))
                      .reduce((a, b) => a + b) /
                  scores.length;
              final stddev = sqrt(variance);
              final cutoff = mean + 1.5 * stddev;
              relevant =
                  aboveFloor.where((r) => (r.score ?? 0) >= cutoff).toList();
            } else {
              relevant = aboveFloor;
            }
            final coverUrl = relevant.isNotEmpty
                ? (relevant.first.thumbUrl.isNotEmpty
                    ? relevant.first.thumbUrl
                    : relevant.first.url)
                : '';

            return CachedCategory(
              name: c['name']!,
              query: c['query']!,
              coverUrl: coverUrl,
              count: relevant.length,
              photoIds: relevant.map((r) => r.id).toList(),
              results: relevant,
            );
          } catch (_) {
            return CachedCategory(
              name: c['name']!,
              query: c['query']!,
              coverUrl: '',
              count: 0,
              photoIds: [],
              results: [],
            );
          }
        }));
        results.addAll(batchResults);
      }

      // Compute diff: new photos per category
      final newPhotoCounts = <String, int>{};
      for (final cat in results) {
        if (cat.count == 0) continue;
        final oldIds = oldPhotoIdsByQuery[cat.query] ?? <int>{};
        final newIds = cat.photoIds.toSet();
        final addedCount = newIds.difference(oldIds).length;
        if (addedCount > 0) {
          newPhotoCounts[cat.name] = addedCount;
        }
      }

      // Save cache
      await saveCache(results);

      return CategoryRefreshResult(
        categories: results,
        newPhotoCounts: newPhotoCounts,
      );
    } finally {
      api.close();
    }
  }
}
