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
      // Surface server error text to the UI for debugging
      throw HttpException('HTTP $statusCode: $body');
    }
    final decoded = jsonDecode(body);
    if (decoded is Map<String, dynamic>) return decoded;
    throw const FormatException('Expected a JSON object at the top level');
  }
}

/// Thin HTTP client for the three backend functions:
/// 1) textSearch  2) imageSearch  3) uploadImages
class ApiClient {
  final http.Client _client;

  /// Optional default headers (e.g., Authorization) you might want to add later.
  final Map<String, String> defaultHeaders;

  ApiClient({http.Client? client, Map<String, String>? defaultHeaders})
      : _client = client ?? http.Client(),
        defaultHeaders = defaultHeaders ?? const {};

  /// Clean up sockets if you ever need to dispose the client.
  /// Clean up sockets if you ever need to dispose the client.
  void close() => _client.close();

  /// HEALTH CHECK
  /// GET /healthz
  /// Returns true if 200 OK
  Future<bool> checkHealth() async {
    try {
      final uri = Uri.parse(AppConfig.healthEndpoint);
      print('[health] GET $uri');
      final resp = await _client.get(uri).timeout(const Duration(seconds: 5));
      print('[health] status=${resp.statusCode}');
      return resp.statusCode == 200;
    } catch (e) {
      print('[health] FAILED: $e');
      return false;
    }
  }

  /// TEXT SEARCH
  /// POST { "user_id": int, "q": "..." } to /api/search/text
  /// Returns SearchResponse { results: [MediaResponse], total: int }
  Future<SearchResponse> textSearch(
    String query, {
    int? limit,
    double? threshold,
  }) async {
    final uri = Uri.parse(AppConfig.textSearchEndpoint);
    final body = <String, dynamic>{
      'user_id': AppConfig.userId,
      'q': query,
    };
    if (limit != null) body['limit'] = limit;
    if (threshold != null) body['threshold'] = threshold;
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
    return SearchResponse.fromJson(resp.requireOkJson());
  }

  /// IMAGE SEARCH
  /// Multipart form-data with fields:
  ///   - user_id : text
  ///   - file    : the probe image file  (NOTE: field name is "file")
  /// Returns SearchResponse.
  Future<SearchResponse> imageSearch(File imageFile, {int limit = 10}) async {
    print('[Search] POST ${AppConfig.imageSearchEndpoint} image=${imageFile.path} user=${AppConfig.userId}');
    final request = http.MultipartRequest('POST', Uri.parse(AppConfig.imageSearchEndpoint));
    request.fields['user_id'] = AppConfig.userId.toString();
    request.fields['limit'] = limit.toString();
    request.files.add(await http.MultipartFile.fromPath('file', imageFile.path));

    final streamedResponse = await request.send();
    final response = await http.Response.fromStream(streamedResponse);

    print('[Search] status=${response.statusCode} body=${response.body}');
    if (response.statusCode == 200) {
      return SearchResponse.fromJson(jsonDecode(response.body));
    }
    throw Exception('Image search failed: ${response.statusCode}');
  }


  /// UPLOAD IMAGES
  /// Multipart form-data with fields:
  ///   - files[] : one or many image files
  ///   - dedup   : "true"/"false"
  /// Returns UploadResponseModel { results:[...], summary:{...} }
  Future<UploadResponseModel> uploadImages({
    required List<File> files,
    required List<String> localPaths,
    bool dedup = true,
  }) async {
    if (files.isEmpty) {
      throw ArgumentError('files list is empty');
    }
    final uri = Uri.parse(AppConfig.uploadEndpoint);
    final req = http.MultipartRequest('POST', uri)
      ..fields['user_id'] = AppConfig.userId.toString()
      ..fields['dedup'] = dedup.toString()
      ..headers.addAll({'Accept': 'application/json', ...defaultHeaders});

    for (final f in files) {
      req.files.add(await http.MultipartFile.fromPath('files[]', f.path));
    }

    for (final path in localPaths) {
      // Use MultipartFile.fromString to send multiple values for the same key (array)
      // req.fields is a Map and overwrites duplicate keys.
      req.files.add(http.MultipartFile.fromString('local_paths[]', path));
    }

    final streamed = await req.send().timeout(const Duration(minutes: 2));
    final resp = await http.Response.fromStream(streamed);
    return UploadResponseModel.fromJson(resp.requireOkJson());
  }

    /// ====== USER IMAGES LIST (for Home gallery) ======
  ///
  /// type UserImagesListRequest struct {
  ///   UserID int64 `json:"user_id" binding:"required"`
  ///   Limit  int   `json:"limit,omitempty"`
  ///   Offset int   `json:"offset,omitempty"`
  /// }
  ///
  /// type UserImagesListResponse struct {
  ///   Results []MediaItem `json:"results"`
  ///   Total   int         `json:"total"`
  /// }
  static Future<List<MediaItem>> listUserImages({
    required int userId,
    int limit = 100,
    int offset = 0,
  }) async {
    // Use GET with query parameters
    final uri = Uri.parse('${AppConfig.baseUrl}/api/user/images/list').replace(
      queryParameters: {
        'user_id': userId.toString(),
        'limit': limit.toString(),
        'offset': offset.toString(),
      },
    );

    final response = await http.get(uri);

        if (response.statusCode != 200) {
      throw Exception(
        'List images failed: ${response.statusCode} - ${response.body}',
      );
    }

    final jsonData = jsonDecode(response.body);
    final listResponse = UserImagesListResponse.fromJson(jsonData);
    return listResponse.results;
  }

  // ====== DELETE IMAGES BATCH ======
  /// DELETE /api/user/images/delete-batch
  /// Removes specific images by their IDs.
  Future<void> deleteImagesBatch(List<int> imageIds) async {
    final uri = Uri.parse(AppConfig.deleteBatchEndpoint);
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

    if (resp.statusCode < 200 || resp.statusCode >= 300) {
      throw HttpException('Delete batch failed: HTTP ${resp.statusCode}: ${resp.body}');
    }
  }

  // ====== CLEAR GALLERY ======
  /// DELETE /api/user/images/clear
  /// Wipes all images for the user but keeps account active.
  Future<void> clearGallery() async {
    final uri = Uri.parse(AppConfig.clearGalleryEndpoint);
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
          }),
        )
        .timeout(const Duration(seconds: 15));

    if (resp.statusCode < 200 || resp.statusCode >= 300) {
      throw HttpException('Clear gallery failed: HTTP ${resp.statusCode}: ${resp.body}');
    }
  }

  // ====== DELETE ACCOUNT ======
  /// DELETE /api/user/account
  /// Irreversibly destroys all user data.
  Future<void> deleteAccount() async {
    final uri = Uri.parse(AppConfig.deleteAccountEndpoint);
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
          }),
        )
        .timeout(const Duration(seconds: 15));

    if (resp.statusCode < 200 || resp.statusCode >= 300) {
      throw HttpException('Delete account failed: HTTP ${resp.statusCode}: ${resp.body}');
    }
  }
}
