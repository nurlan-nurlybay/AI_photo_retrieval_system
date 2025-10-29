// lib/data/api_client.dart
import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;

import '../core/config.dart';
import 'models.dart';
import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/data/models.dart';

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
  void close() => _client.close();

  /// TEXT SEARCH
  /// POST { "user_id": int, "q": "..." } to /api/search/text
  /// Returns SearchResponse { results: [MediaResponse], total: int }
  Future<SearchResponse> textSearch(String query) async {
    final uri = Uri.parse(AppConfig.textSearchEndpoint);
    final resp = await _client
        .post(
          uri,
          headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json',
            ...defaultHeaders,
          },
          body: jsonEncode({'user_id': AppConfig.userId, 'q': query}),
        )
        .timeout(const Duration(seconds: 20));
    return SearchResponse.fromJson(resp.requireOkJson());
  }

  /// IMAGE SEARCH
  /// Multipart form-data with fields:
  ///   - user_id : text
  ///   - file    : the probe image file  (NOTE: field name is "file")
  /// Returns SearchResponse.
Future<SearchResponse> imageSearch(File probeImage) async {
  final uri = Uri.parse(AppConfig.imageSearchEndpoint);
  final req = http.MultipartRequest('POST', uri)
    ..fields['user_id'] = AppConfig.userId.toString()
    ..files.add(await http.MultipartFile.fromPath('file', probeImage.path))
    ..headers['Accept'] = 'application/json';         // ok
    // DO NOT set 'Content-Type' here.

  // Debug logs:
  print('-> POST $uri');
  print('   fields: ${req.fields}');
  print('   file exists: ${await probeImage.exists()} path: ${probeImage.path}');

  final streamed = await req.send();
  final resp = await http.Response.fromStream(streamed);
  print('<- ${resp.statusCode} ${resp.headers['content-type']}');
  print(resp.body);
  return SearchResponse.fromJson(jsonDecode(resp.body));
}


  /// UPLOAD IMAGES
  /// Multipart form-data with fields:
  ///   - files[] : one or many image files
  ///   - dedup   : "true"/"false"
  /// Returns UploadResponseModel { results:[...], summary:{...} }
  Future<UploadResponseModel> uploadImages({
    required List<File> files,
    bool dedup = true,
  }) async {
    if (files.isEmpty) {
      throw ArgumentError('files list is empty');
    }
    final uri = Uri.parse(AppConfig.uploadEndpoint);
    final req = http.MultipartRequest('POST', uri)
      ..fields['dedup'] = dedup.toString()
      ..headers.addAll({'Accept': 'application/json', ...defaultHeaders});

    for (final f in files) {
      req.files.add(await http.MultipartFile.fromPath('files[]', f.path));
    }

    final streamed = await req.send().timeout(const Duration(minutes: 2));
    final resp = await http.Response.fromStream(streamed);
    return UploadResponseModel.fromJson(resp.requireOkJson());
  }
}
