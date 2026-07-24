import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config.dart';
import '../models/position_update.dart';

/// Wraps GET /ws/trips/{id} (see backend/internal/ws/gateway.go) as a Dart Stream.
/// Token travels as a query parameter, not a header - browsers' WebSocket API
/// can't set custom headers on the handshake (backend/internal/httpapi/middleware.go
/// RequireAuthQuery exists specifically for this).
class TripSocket {
  WebSocketChannel? _channel;

  Stream<PositionUpdate> connect(int tripId, String token) {
    final uri = Uri.parse('$wsBaseUrl/ws/trips/$tripId').replace(queryParameters: {'token': token});
    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    return channel.stream.map((raw) => PositionUpdate.fromJson(jsonDecode(raw as String) as Map<String, dynamic>));
  }

  void close() {
    _channel?.sink.close();
    _channel = null;
  }
}
