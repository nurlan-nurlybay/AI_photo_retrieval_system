// lib/data/api_client.dart
import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;

import '../core/config.dart';
import 'models.dart';

/// Convenience extension to validate and decode JSON responses.
extension _HttpResponseX on http.Response {
  Map<String, dynamic> requireOkJson() {
    if (statusCode < 200 || statusCode >= 300) {
      throw HttpException('HTTP $statusCode: $body');
    }
    final decoded = jsonDecode(body);
    if (decoded is Map<String, dynamic>) return decoded;
    throw const FormatException('Expected a JSON object at the top level');
  }
}

/// Structured logger for API calls with timestamps and duration tracking.
class _Log {
  static String _ts() => DateTime.now().toIso8601String().substring(11, 23);
  
  static void info(String tag, String msg) => print('[${_ts()}] 📡 [$tag] $msg');
  static void ok(String tag, String msg) => print('[${_ts()}] ✅ [$tag] $msg');
  static void warn(String tag, String msg) => print('[${_ts()}] ⚠️  [$tag] $msg');
  static void fail(String tag, String msg) => print('[${_ts()}] ❌ [$tag] $msg');
  static void data(String tag, String msg) => print('[${_ts()}] 📦 [$tag] $msg');
}

/// Thin HTTP client for the backend API.
class ApiClient {
  final http.Client _client;
  final Map<String, String> defaultHeaders;

  ApiClient({http.Client? client, Map<String, String>? defaultHeaders})
      : _client = client ?? http.Client(),
        defaultHeaders = defaultHeaders ?? const {};

  void close() => _client.close();

  // ====== HEALTH CHECK ======
  Future<bool> checkHealth() async {
    const tag = 'HEALTH';
    try {
      final uri = Uri.parse(AppConfig.healthEndpoint);
      _Log.info(tag, 'GET $uri');
      final sw = Stopwatch()..start();
      final resp = await _client.get(uri).timeout(const Duration(seconds: 5));
      sw.stop();
      _Log.ok(tag, 'status=${resp.statusCode} (${sw.elapsedMilliseconds}ms)');
      return resp.statusCode == 200;
    } catch (e) {
      _Log.fail(tag, '$e');
      return false;
    }
  }

  // ====== TEXT SEARCH ======
  Future<SearchResponse> textSearch(
    String query, {
    int? limit,
    double? threshold,
  }) async {
    const tag = 'TEXT_SEARCH';
    final uri = Uri.parse(AppConfig.textSearchEndpoint);
    final body = <String, dynamic>{
      'user_id': AppConfig.userId,
      'q': query,
    };
    if (limit != null) body['limit'] = limit;
    if (threshold != null) body['threshold'] = threshold;

    _Log.info(tag, 'POST $uri');
    _Log.data(tag, 'payload: ${jsonEncode(body)}');
    final sw = Stopwatch()..start();

    try {
      final resp = await _client
          .post(
            uri,
            headers: {
              'Content-Type': 'application/json',
              'Accept': 'application/json',
              ...defaultHeaders,
            },
            body: jsonEncode(body),
          )
          .timeout(const Duration(seconds: 120));
      sw.stop();

      _Log.ok(tag, 'status=${resp.statusCode} (${sw.elapsedMilliseconds}ms)');
      _Log.data(tag, 'body=${resp.body.length > 500 ? '${resp.body.substring(0, 500)}...' : resp.body}');

      final parsed = SearchResponse.fromJson(resp.requireOkJson());
      _Log.ok(tag, 'parsed: ${parsed.results.length} results, total=${parsed.total}, usedQwen=${parsed.usedQwen}');
      for (var i = 0; i < parsed.results.length; i++) {
        final r = parsed.results[i];
        _Log.data(tag, '  [$i] id=${r.id} score=${r.score} status=${r.status.name} url=${r.url.substring(0, 60.clamp(0, r.url.length))}...');
      }
      return parsed;
    } catch (e, st) {
      sw.stop();
      _Log.fail(tag, 'FAILED after ${sw.elapsedMilliseconds}ms: $e');
      _Log.fail(tag, 'stacktrace: ${st.toString().split('\n').take(5).join(' | ')}');
      rethrow;
    }
  }

  // ====== IMAGE SEARCH ======
  Future<SearchResponse> imageSearch(File imageFile, {int limit = 10}) async {
    const tag = 'IMG_SEARCH';
    final uri = Uri.parse(AppConfig.imageSearchEndpoint);
    final fileSize = await imageFile.length();

    _Log.info(tag, 'POST $uri');
    _Log.data(tag, 'file=${imageFile.path}');
    _Log.data(tag, 'fileSize=${(fileSize / 1024).toStringAsFixed(1)}KB, user=${AppConfig.userId}, limit=$limit');
    _Log.info(tag, '⏳ Sending multipart request...');

    final sw = Stopwatch()..start();

    try {
      final request = http.MultipartRequest('POST', uri);
      request.fields['user_id'] = AppConfig.userId.toString();
      request.fields['limit'] = limit.toString();
      request.files.add(await http.MultipartFile.fromPath('file', imageFile.path));

      _Log.info(tag, '⏳ Awaiting server response... (timeout: 120s)');
      final streamedResponse = await request.send().timeout(const Duration(seconds: 120));
      _Log.info(tag, '⏳ Reading response body...');
      final response = await http.Response.fromStream(streamedResponse);
      sw.stop();

      _Log.ok(tag, 'status=${response.statusCode} (${sw.elapsedMilliseconds}ms)');
      _Log.data(tag, 'body=${response.body.length > 500 ? '${response.body.substring(0, 500)}...' : response.body}');

      if (response.statusCode == 200) {
        final parsed = SearchResponse.fromJson(jsonDecode(response.body));
        _Log.ok(tag, 'parsed: ${parsed.results.length} results, total=${parsed.total}');
        for (var i = 0; i < parsed.results.length; i++) {
          final r = parsed.results[i];
          _Log.data(tag, '  [$i] id=${r.id} score=${r.score} status=${r.status.name}');
        }
        return parsed;
      }
      throw Exception('Image search failed: ${response.statusCode} ${response.body}');
    } catch (e, st) {
      sw.stop();
      _Log.fail(tag, 'FAILED after ${sw.elapsedMilliseconds}ms: $e');
      _Log.fail(tag, 'stacktrace: ${st.toString().split('\n').take(5).join(' | ')}');
      rethrow;
    }
  }

  // ====== UPLOAD IMAGES ======
  Future<UploadResponseModel> uploadImages({
    required List<File> files,
    required List<String> localPaths,
    bool dedup = true,
  }) async {
    const tag = 'UPLOAD';
    if (files.isEmpty) {
      throw ArgumentError('files list is empty');
    }
    final uri = Uri.parse(AppConfig.uploadEndpoint);

    _Log.info(tag, 'POST $uri');
    _Log.data(tag, 'files=${files.length}, dedup=$dedup, user=${AppConfig.userId}');
    final sw = Stopwatch()..start();

    try {
      final req = http.MultipartRequest('POST', uri)
        ..fields['user_id'] = AppConfig.userId.toString()
        ..fields['dedup'] = dedup.toString()
        ..headers.addAll({'Accept': 'application/json', ...defaultHeaders});

      for (final f in files) {
        req.files.add(await http.MultipartFile.fromPath('files[]', f.path));
      }
      for (final path in localPaths) {
        req.files.add(http.MultipartFile.fromString('local_paths[]', path));
      }

      _Log.info(tag, '⏳ Sending ${files.length} file(s)...');
      final streamed = await req.send().timeout(const Duration(minutes: 2));
      final resp = await http.Response.fromStream(streamed);
      sw.stop();

      _Log.ok(tag, 'status=${resp.statusCode} (${sw.elapsedMilliseconds}ms)');
      _Log.data(tag, 'body=${resp.body.length > 300 ? '${resp.body.substring(0, 300)}...' : resp.body}');

      final parsed = UploadResponseModel.fromJson(resp.requireOkJson());
      _Log.ok(tag, 'summary: saved=${parsed.summary.saved}, dups=${parsed.summary.duplicates}, failed=${parsed.summary.failed}');
      return parsed;
    } catch (e, st) {
      sw.stop();
      _Log.fail(tag, 'FAILED after ${sw.elapsedMilliseconds}ms: $e');
      _Log.fail(tag, 'stacktrace: ${st.toString().split('\n').take(3).join(' | ')}');
      rethrow;
    }
  }

  // ====== USER IMAGES LIST ======
  static Future<List<MediaItem>> listUserImages({
    required int userId,
    int limit = 100,
    int offset = 0,
  }) async {
    const tag = 'LIST_IMAGES';
    final uri = Uri.parse('${AppConfig.baseUrl}/api/user/images/list').replace(
      queryParameters: {
        'user_id': userId.toString(),
        'limit': limit.toString(),
        'offset': offset.toString(),
      },
    );

    _Log.info(tag, 'GET $uri');
    final sw = Stopwatch()..start();

    final response = await http.get(uri);
    sw.stop();
    _Log.ok(tag, 'status=${response.statusCode} (${sw.elapsedMilliseconds}ms)');

    if (response.statusCode != 200) {
      _Log.fail(tag, 'body=${response.body}');
      throw Exception('List images failed: ${response.statusCode} - ${response.body}');
    }

    final jsonData = jsonDecode(response.body);
    final listResponse = UserImagesListResponse.fromJson(jsonData);
    _Log.ok(tag, 'returned ${listResponse.results.length} items (total=${listResponse.total})');
    return listResponse.results;
  }

  // ====== DELETE IMAGES BATCH ======
  Future<void> deleteImagesBatch(List<int> imageIds) async {
    const tag = 'DELETE_BATCH';
    final uri = Uri.parse(AppConfig.deleteBatchEndpoint);

    _Log.info(tag, 'DELETE $uri ids=$imageIds user=${AppConfig.userId}');
    final sw = Stopwatch()..start();

    try {
      final resp = await _client
          .delete(
            uri,
            headers: {
              'Content-Type': 'application/json',
              'Accept': 'application/json',
              ...defaultHeaders,
            },
            body: jsonEncode({
              'user_id': AppConfig.userId,
              'image_ids': imageIds,
            }),
          )
          .timeout(const Duration(seconds: 15));
      sw.stop();

      _Log.ok(tag, 'status=${resp.statusCode} (${sw.elapsedMilliseconds}ms) body=${resp.body}');
      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        throw HttpException('Delete batch failed: HTTP ${resp.statusCode}: ${resp.body}');
      }
    } catch (e) {
      _Log.fail(tag, '$e');
      rethrow;
    }
  }

  // ====== CLEAR GALLERY ======
  Future<void> clearGallery() async {
    const tag = 'CLEAR_GALLERY';
    final uri = Uri.parse(AppConfig.clearGalleryEndpoint);

    _Log.info(tag, 'DELETE $uri user=${AppConfig.userId}');
    final sw = Stopwatch()..start();

    try {
      final resp = await _client
          .delete(
            uri,
            headers: {
              'Content-Type': 'application/json',
              'Accept': 'application/json',
              ...defaultHeaders,
            },
            body: jsonEncode({'user_id': AppConfig.userId}),
          )
          .timeout(const Duration(seconds: 15));
      sw.stop();

      _Log.ok(tag, 'status=${resp.statusCode} (${sw.elapsedMilliseconds}ms) body=${resp.body}');
      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        throw HttpException('Clear gallery failed: HTTP ${resp.statusCode}: ${resp.body}');
      }
    } catch (e) {
      _Log.fail(tag, '$e');
      rethrow;
    }
  }

  // ====== DELETE ACCOUNT ======
  Future<void> deleteAccount() async {
    const tag = 'NUKE_ACCOUNT';
    final uri = Uri.parse(AppConfig.deleteAccountEndpoint);

    _Log.warn(tag, '💀 DELETE $uri user=${AppConfig.userId}');
    final sw = Stopwatch()..start();

    try {
      final resp = await _client
          .delete(
            uri,
            headers: {
              'Content-Type': 'application/json',
              'Accept': 'application/json',
              ...defaultHeaders,
            },
            body: jsonEncode({'user_id': AppConfig.userId}),
          )
          .timeout(const Duration(seconds: 15));
      sw.stop();

      _Log.ok(tag, 'status=${resp.statusCode} (${sw.elapsedMilliseconds}ms) body=${resp.body}');
      if (resp.statusCode < 200 || resp.statusCode >= 300) {
        throw HttpException('Delete account failed: HTTP ${resp.statusCode}: ${resp.body}');
      }
    } catch (e) {
      _Log.fail(tag, '$e');
      rethrow;
    }
  }
}
