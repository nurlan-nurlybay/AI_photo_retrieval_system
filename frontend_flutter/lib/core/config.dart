import 'dart:io';
import 'package:frontend_flutter/data/models.dart';

/// Central configuration for environment + endpoints.
/// Edit here, not all over the codebase.
class AppConfig {
  /// 1) Base URL — Production backend on AWS EC2 (Stockholm, eu-north-1).
  ///    Override at run-time with:
  ///    flutter run ... --dart-define=BACKEND_BASE_URL=http://13.61.195.243
  static const String _rawBaseUrl =
      String.fromEnvironment('BACKEND_BASE_URL', defaultValue: 'http://13.61.195.243');

  /// 2) Normalize addresses per platform.
  ///    Since the backend is now a public IP, platform rewrites are only
  ///    needed when someone explicitly passes localhost/10.0.2.2 for local dev.
  static String get baseUrl {
    if (Platform.isAndroid) {
      if (_rawBaseUrl.contains('127.0.0.1') || _rawBaseUrl.contains('localhost')) {
        return _rawBaseUrl
            .replaceFirst('127.0.0.1', '10.0.2.2')
            .replaceFirst('localhost', '10.0.2.2');
      }
    } else if (Platform.isIOS) {
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

  // ─── Endpoints ─────────────────────────────────────────────────────

  /// Upload images
  static String get uploadEndpoint          => _u('/api/upload/image');

  /// Text search
  static String get textSearchEndpoint      => _u('/api/search/text');

  /// Image (reverse) search
  static String get imageSearchEndpoint     => _u('/api/search/image');

  /// Health check
  static String get healthEndpoint          => _u('/healthz');

  /// SSE status stream (real-time processing updates via Redis Pub/Sub)
  static String get statusStreamEndpoint    => _u('/api/status-stream');

  /// Granular delete (batch)
  static String get deleteBatchEndpoint     => _u('/api/user/images/delete-batch');

  /// Clear gallery (wipe all images, keep account)
  static String get clearGalleryEndpoint    => _u('/api/user/images/clear');

  /// Nuke account (irreversible)
  static String get deleteAccountEndpoint   => _u('/api/user/account');

  /// User images list
  static String get userImagesListEndpoint  => _u('/api/user/images/list');

  // ─── Image URL helpers ─────────────────────────────────────────────

  /// Fix image URLs that may contain Docker-internal or localhost hostnames.
  /// In production the backend returns URLs relative to the public IP, so
  /// this is mainly a safety-net for legacy data.
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

  // ─── User ──────────────────────────────────────────────────────────

  /// Temporary user id for MVP (replace with real auth later).
  /// Changed to 1 to match the multi-tenant API examples.
  static const int userId = 500;
}
