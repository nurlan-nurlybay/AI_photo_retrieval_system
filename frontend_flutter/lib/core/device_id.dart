import 'dart:io';

import 'package:device_info_plus/device_info_plus.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Generates and persists a stable, device-unique user ID.
///
/// Source of truth (in priority order):
///   iOS  → identifierForVendor (IDFV)
///   Android → androidId
///   Fallback → random timestamp (stored permanently in SharedPreferences)
///
/// The raw string is hashed with a simple FNV-1a-style function to produce
/// a positive int64 that is consistent across app restarts.
class DeviceIdService {
  static const _prefKey = 'device_user_id';

  /// Returns the device-unique user ID.
  /// On first call it derives, hashes, and persists the value.
  /// Subsequent calls return the value from SharedPreferences immediately.
  static Future<int> getUserId() async {
    final prefs = await SharedPreferences.getInstance();

    final cached = prefs.getInt(_prefKey);
    if (cached != null && cached > 0) return cached;

    final raw = await _rawDeviceId();
    final id = _stableHash(raw);
    await prefs.setInt(_prefKey, id);
    return id;
  }

  // ── internals ────────────────────────────────────────────────────────────

  static Future<String> _rawDeviceId() async {
    try {
      final plugin = DeviceInfoPlugin();
      if (Platform.isIOS) {
        final ios = await plugin.iosInfo;
        final idfv = ios.identifierForVendor;
        if (idfv != null && idfv.isNotEmpty) return idfv;
      } else if (Platform.isAndroid) {
        final android = await plugin.androidInfo;
        final aid = android.id;
        if (aid.isNotEmpty) return aid;
      }
    } catch (_) {
      // Ignore — fall through to timestamp fallback
    }
    // Should never happen on real iOS/Android devices.
    return 'fallback-${DateTime.now().microsecondsSinceEpoch}';
  }

  /// Deterministic hash of [s] → positive int64.
  /// djb2-style polynomial hash, masked to 63 bits so the result is always
  /// non-negative and fits both a Dart native int and a Go int64.
  static int _stableHash(String s) {
    var h = 5381;
    for (final c in s.codeUnits) {
      h = (h * 31 + c) & 0x7FFFFFFFFFFFFFFF;
    }
    return h == 0 ? 1 : h;
  }
}
