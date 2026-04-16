import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:frontend_flutter/core/config.dart';
import 'package:frontend_flutter/data/models.dart';

/// Event emitted by the status stream.
class StatusUpdate {
  final int mediaId;
  final int userId;
  final ImageProcessingStatus status;

  StatusUpdate({
    required this.mediaId,
    required this.userId,
    required this.status,
  });

  factory StatusUpdate.fromJson(Map<String, dynamic> json) {
    return StatusUpdate(
      mediaId: json['media_id'] as int,
      userId: json['user_id'] as int,
      status: ImageProcessingStatus.fromString(json['status'] as String?),
    );
  }
}

/// Singleton service that manages the Server-Sent Events (SSE) connection
/// to receive real-time image processing updates.
class StatusStreamService {
  StatusStreamService._();
  static final StatusStreamService instance = StatusStreamService._();

  final StreamController<StatusUpdate> _controller =
      StreamController<StatusUpdate>.broadcast();

  Stream<StatusUpdate> get stream => _controller.stream;

  http.Client? _client;
  bool _isConnected = false;
  bool _shouldReconnect = true;
  int _reconnectAttempts = 0;

  static String _ts() => DateTime.now().toIso8601String().substring(11, 23);

  /// Start the SSE connection.
  void start() {
    if (_isConnected) {
      print('[${_ts()}] 📡 [SSE] Already connected, skipping start()');
      return;
    }
    print('[${_ts()}] 📡 [SSE] start() called');
    _shouldReconnect = true;
    _reconnectAttempts = 0;
    _connect();
  }

  /// Stop the SSE connection.
  void stop() {
    print('[${_ts()}] 📡 [SSE] stop() called');
    _shouldReconnect = false;
    _client?.close();
    _isConnected = false;
  }

  Future<void> _connect() async {
    _isConnected = true;
    _client = http.Client();
    final url = AppConfig.statusStreamEndpoint;

    print('[${_ts()}] 📡 [SSE] Connecting to $url (attempt #${_reconnectAttempts + 1})');

    try {
      final request = http.Request('GET', Uri.parse(url));
      final sw = Stopwatch()..start();
      final response = await _client!.send(request).timeout(const Duration(minutes: 5));
      sw.stop();

      print('[${_ts()}] 📡 [SSE] Connected! status=${response.statusCode} (${sw.elapsedMilliseconds}ms)');

      if (response.statusCode != 200) {
        print('[${_ts()}] 📡 [SSE] Bad status code: ${response.statusCode}');
        throw Exception('SSE status ${response.statusCode}');
      }

      _reconnectAttempts = 0; // Reset on successful connect

      // Read the stream line by line
      response.stream
          .transform(utf8.decoder)
          .transform(const LineSplitter())
          .listen(
        (line) {
          if (line.startsWith('data: ')) {
            final data = line.substring(6).trim();
            if (data.isEmpty || data == 'connected') {
              print('[${_ts()}] 📡 [SSE] Handshake/heartbeat: "$data"');
              return;
            }
            try {
              final json = jsonDecode(data);
              final update = StatusUpdate.fromJson(json);
              print('[${_ts()}] 📡 [SSE] ✅ Event: mediaId=${update.mediaId} status=${update.status.name}');
              _controller.add(update);
            } catch (e) {
              print('[${_ts()}] 📡 [SSE] ⚠️ Decode error: $e (raw: "${data.length > 100 ? '${data.substring(0, 100)}...' : data}")');
            }
          } else if (line.isNotEmpty) {
            print('[${_ts()}] 📡 [SSE] Non-data line: "$line"');
          }
        },
        onError: (e) {
          print('[${_ts()}] 📡 [SSE] ❌ Stream error: $e');
          _reconnect();
        },
        onDone: () {
          print('[${_ts()}] 📡 [SSE] Stream closed (onDone)');
          _reconnect();
        },
        cancelOnError: true,
      );
    } catch (e) {
      print('[${_ts()}] 📡 [SSE] ❌ Connection failed: $e');
      _reconnect();
    }
  }

  void _reconnect() {
    _isConnected = false;
    _client?.close();
    if (_shouldReconnect) {
      _reconnectAttempts++;
      // Exponential backoff: 5s, 10s, 20s, 40s... max 60s
      final delay = Duration(seconds: (5 * (1 << (_reconnectAttempts - 1).clamp(0, 3))).clamp(5, 60));
      print('[${_ts()}] 📡 [SSE] Reconnecting in ${delay.inSeconds}s (attempt #$_reconnectAttempts)');
      Future.delayed(delay, () {
        if (_shouldReconnect && !_isConnected) {
          _connect();
        }
      });
    } else {
      print('[${_ts()}] 📡 [SSE] Reconnect disabled, staying disconnected');
    }
  }
}
