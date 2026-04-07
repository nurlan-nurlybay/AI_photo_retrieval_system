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

  /// Start the SSE connection.
  void start() {
    if (_isConnected) return;
    _shouldReconnect = true;
    _connect();
  }

  /// Stop the SSE connection.
  void stop() {
    _shouldReconnect = false;
    _client?.close();
    _isConnected = false;
  }

  Future<void> _connect() async {
    _isConnected = true;
    _client = http.Client();

    try {
      final request = http.Request('GET', Uri.parse(AppConfig.statusStreamEndpoint));
      // SSE usually requires keeping the connection open
      final response = await _client!.send(request).timeout(const Duration(minutes: 5));

      if (response.statusCode != 200) {
        throw Exception('SSE status ${response.statusCode}');
      }

      // Read the stream line by line
      response.stream
          .transform(utf8.decoder)
          .transform(const LineSplitter())
          .listen(
        (line) {
          if (line.startsWith('data: ')) {
            final data = line.substring(6).trim();
            if (data.isEmpty || data == 'connected') return;
            try {
              final json = jsonDecode(data);
              _controller.add(StatusUpdate.fromJson(json));
            } catch (e) {
              // Silently ignore if it's just a non-JSON handshake or heartbeat
              if (data != 'connected') {
                print('[SSE] Error decoding data: $e (Raw: $data)');
              }
            }
          }
        },
        onError: (e) {
          print('[SSE] Stream error: $e');
          _reconnect();
        },
        onDone: () {
          print('[SSE] Stream closed');
          _reconnect();
        },
        cancelOnError: true,
      );
    } catch (e) {
      print('[SSE] Connection failed: $e');
      _reconnect();
    }
  }

  void _reconnect() {
    _isConnected = false;
    _client?.close();
    if (_shouldReconnect) {
      // Linear backoff for simplicity
      Future.delayed(const Duration(seconds: 5), () {
        if (_shouldReconnect && !_isConnected) {
          _connect();
        }
      });
    }
  }
}
