import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config.dart';
import '../models/position_update.dart';

/// Wraps GET /ws/trips/{id} (see backend/internal/ws/gateway.go) as a Dart Stream.
/// Token travels as a query parameter, not a header - browsers' WebSocket API
/// can't set custom headers on the handshake (backend/internal/httpapi/middleware.go
/// RequireAuthQuery exists specifically for this).
///
/// Reconnects automatically (with backoff) on drop - live GPS tracking makes a
/// dropped connection more costly than in the pure-simulation days, since a
/// dispatcher watching a real truck shouldn't lose the feed over a brief
/// network hiccup.
class TripSocket {
  static const _retryDelays = [Duration(seconds: 1), Duration(seconds: 2), Duration(seconds: 5), Duration(seconds: 10)];

  WebSocketChannel? _channel;
  StreamController<PositionUpdate>? _controller;
  int? _tripId;
  String? _token;
  bool _closed = false;
  int _retryCount = 0;

  Stream<PositionUpdate> connect(int tripId, String token) {
    _tripId = tripId;
    _token = token;
    _closed = false;
    _controller = StreamController<PositionUpdate>.broadcast(onCancel: close);
    _open();
    return _controller!.stream;
  }

  void _open() {
    final uri = Uri.parse('$wsBaseUrl/ws/trips/$_tripId').replace(queryParameters: {'token': _token});
    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    channel.stream.listen(
      (raw) {
        _retryCount = 0;
        final update = PositionUpdate.fromJson(jsonDecode(raw as String) as Map<String, dynamic>);
        _controller?.add(update);
        if (update.status == 'arrived') close();
      },
      onError: (_) => _scheduleReconnect(),
      onDone: _scheduleReconnect,
      cancelOnError: true,
    );
  }

  void _scheduleReconnect() {
    if (_closed || _controller == null || _controller!.isClosed) return;
    final delay = _retryDelays[_retryCount.clamp(0, _retryDelays.length - 1)];
    _retryCount++;
    Future.delayed(delay, () {
      if (!_closed) _open();
    });
  }

  void close() {
    _closed = true;
    _channel?.sink.close();
    _channel = null;
    _controller?.close();
    _controller = null;
  }
}
