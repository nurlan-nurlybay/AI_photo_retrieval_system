import 'dart:convert';
import 'dart:io';
import 'package:http/http.dart' as http;

class ApiService {
  static const String baseUrl = 'http://10.0.2.2:8080'; // use this for emulator
  // For real device: replace with your machine IP (e.g. 192.168.x.x)

  // Upload up to 10 images
  static Future<Map<String, dynamic>> uploadImages(List<File> images) async {
    var uri = Uri.parse('$baseUrl/api/upload/image');
    var request = http.MultipartRequest('POST', uri);

    for (var image in images.take(10)) {
      request.files.add(await http.MultipartFile.fromPath('images', image.path));
    }

    var response = await request.send();
    var resBody = await response.stream.bytesToString();

    if (response.statusCode == 200) {
      return jsonDecode(resBody);
    } else {
      throw Exception('Failed to upload images: ${response.statusCode}');
    }
  }

  // Text-based search
  static Future<List<dynamic>> searchText(int userId, String query) async {
    final uri = Uri.parse('$baseUrl/api/search/text');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'user_id': userId, 'q': query}),
    );

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      return data['data'];
    } else {
      throw Exception('Search failed: ${response.statusCode}');
    }
  }
}
