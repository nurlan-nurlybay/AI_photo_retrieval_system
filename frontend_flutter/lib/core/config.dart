import 'dart:io';
import 'package:frontend_flutter/data/models.dart';

/// Central configuration for environment + endpoints.
/// Edit here, not all over the codebase.
class AppConfig {
  /// 1) Base URL provided at run-time with:
  /// flutter run ... --dart-define=BACKEND_BASE_URL=http://127.0.0.1:8080
  static const String _rawBaseUrl =
      String.fromEnvironment('BACKEND_BASE_URL', defaultValue: 'http://10.0.2.2:8080');

  /// 2) Normalize localhost for Android emulator (10.0.2.2)
  static String get baseUrl {
    if (Platform.isAndroid && _rawBaseUrl.contains('127.0.0.1')) {
      // Mirrored Networking: 10.0.2.2 works correctly!
      return _rawBaseUrl.replaceFirst('127.0.0.1', '10.0.2.2');
    }
    return _rawBaseUrl;
  }

  /// 3) Helper to safely join base + path (avoids double slashes)
  static String _u(String path) {
    final b = baseUrl.endsWith('/') ? baseUrl.substring(0, baseUrl.length - 1) : baseUrl;
    final p = path.startsWith('/') ? path : '/$path';
    return '$b$p';
  }

  /// 4) Endpoints (adjust paths if your backend differs)
  static String get uploadEndpoint      => _u('/api/upload/image');
  static String get textSearchEndpoint  => _u('/api/search/text');
  static String get imageSearchEndpoint => _u('/api/search/image');
  static String get healthEndpoint => '$baseUrl/healthz';

  /// Helper to fix image URLs for Android Emulator
  /// In Mirrored Mode, SeaweedFS (port 8333) needs the actual WSL IP
  static String fixImageUrl(String url) {
    if (Platform.isAndroid && url.contains('localhost:8333')) {
      // Use WSL IP directly (update this if your WSL IP changes)
      return url.replaceFirst('localhost:8333', '192.168.0.160:8333');
    }
    return url;
  }

  /// 5) Temporary user id for MVP (replace with real auth later)
  static const int userId = 404;
}
