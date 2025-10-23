import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;
import '../models/api_models.dart';

class ApiService {
  // Change this to your backend URL
  // For Android emulator: use 10.0.2.2
  // For Linux desktop: use localhost
  // For real device: use your computer's IP address
  static const String baseUrl = 'http://localhost:8080';

  // For testing on real device, uncomment and replace with your IP:
  // static const String baseUrl = 'http://192.168.1.100:8080';

  static const int userId = 404; // Fixed user ID for testing

  /// Upload up to 10 images to the backend
  static Future<UploadResponse> uploadImages(List<File> images) async {
    if (images.isEmpty) {
      throw Exception('No images selected');
    }

    if (images.length > 10) {
      throw Exception('Maximum 10 images allowed');
    }

    final uri = Uri.parse('$baseUrl/api/upload/image');
    final request = http.MultipartRequest('POST', uri);

    // Add each image file to the request
    for (int i = 0; i < images.length; i++) {
      final file = images[i];
      final multipartFile = await http.MultipartFile.fromPath(
        'files[]', // Backend expects 'files[]' parameter name
        file.path,
      );
      request.files.add(multipartFile);
    }

    try {
      final streamedResponse = await request.send();
      final response = await http.Response.fromStream(streamedResponse);

      if (response.statusCode == 200) {
        final jsonData = jsonDecode(response.body);
        return UploadResponse.fromJson(jsonData);
      } else {
        throw Exception(
            'Upload failed: ${response.statusCode} - ${response.body}');
      }
    } catch (e) {
      throw Exception('Upload error: $e');
    }
  }

  /// Search for images using text query
  static Future<List<SearchResult>> searchText(
    String query, {
    required int userId,
  }) async {
    if (query.trim().isEmpty) {
      throw Exception('Search query cannot be empty');
    }

    final uri = Uri.parse('$baseUrl/api/search/text');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'user_id': userId,
        'q': query.trim(),
      }),
    );

    if (response.statusCode != 200) {
      throw Exception(
          'Search failed: ${response.statusCode} - ${response.body}');
    }

    final jsonData = jsonDecode(response.body);
    final searchResponse = SearchResponse.fromJson(jsonData);
    return searchResponse.data;
  }

  /// Search for images using an uploaded image
  static Future<List<SearchResult>> searchByImage(
    File image, {
    required int userId,
  }) async {
    if (!await image.exists()) {
      throw Exception('Image file does not exist');
    }

    final uri = Uri.parse('$baseUrl/api/search/image');

    final request = http.MultipartRequest('POST', uri);

    // Add user_id as form field (required by backend)
    request.fields['user_id'] = userId.toString();

    // Add the image file with correct key name: 'image'
    final multipartFile = await http.MultipartFile.fromPath(
      'image', // must match `form:"image"` in Go struct
      image.path,
    );
    request.files.add(multipartFile);

    try {
      final streamedResponse = await request.send();
      final response = await http.Response.fromStream(streamedResponse);

      if (response.statusCode != 200) {
        throw Exception(
            'Image search failed: ${response.statusCode} - ${response.body}');
      }

      final jsonData = jsonDecode(response.body);
      final searchResponse = SearchResponse.fromJson(jsonData);
      return searchResponse.data;
    } catch (e) {
      throw Exception('Image search error: $e');
    }
  }

  /// Test if the backend is reachable
  static Future<bool> testConnection() async {
    try {
      final uri = Uri.parse('$baseUrl/healthz');
      final response = await http.get(uri).timeout(
            const Duration(seconds: 5),
          );
      return response.statusCode == 200;
    } catch (e) {
      return false;
    }
  }
}
