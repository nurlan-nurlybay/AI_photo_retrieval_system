import 'dart:io';
import 'package:frontend_flutter/data/models.dart';

/// Central configuration for environment + endpoints.
/// Edit here, not all over the codebase.
class AppConfig {
  /// 1) Base URL provided at run-time with:
  /// flutter run ... --dart-define=BACKEND_BASE_URL=http://127.0.0.1:8080
  static const String _rawBaseUrl =
      String.fromEnvironment('BACKEND_BASE_URL', defaultValue: 'http://10.0.2.2:8080');

  /// 2) Normalize addresses per platform
  static String get baseUrl {
    if (Platform.isAndroid) {
      // Android emulator: 10.0.2.2 maps to host localhost
      if (_rawBaseUrl.contains('127.0.0.1') || _rawBaseUrl.contains('localhost')) {
        return _rawBaseUrl
            .replaceFirst('127.0.0.1', '10.0.2.2')
            .replaceFirst('localhost', '10.0.2.2');
      }
    } else if (Platform.isIOS) {
      // iOS simulator: 10.0.2.2 doesn't work, use localhost
      if (_rawBaseUrl.contains('10.0.2.2')) {
        return _rawBaseUrl.replaceFirst('10.0.2.2', 'localhost');
      }
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
    // Legacy: old photos stored with Docker-internal hostname
    if (url.contains('seaweedfs:8888')) {
      url = url.replaceFirst('seaweedfs:8888', 'localhost:8888');
    }
    // Android emulator can't reach host via localhost
    if (Platform.isAndroid && url.contains('localhost:8888')) {
      return url.replaceFirst('localhost:8888', '10.0.2.2:8888');
    }
    return url;
  }

  /// 5) Temporary user id for MVP (replace with real auth later)
  static const int userId = 404;
}
