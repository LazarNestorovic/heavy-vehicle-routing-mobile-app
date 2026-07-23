import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config.dart';
import '../models/position_update.dart';

/// Wraps GET /ws/trips/{id} (see backend/internal/ws/gateway.go) as a Dart Stream.
class TripSocket {
  WebSocketChannel? _channel;

  Stream<PositionUpdate> connect(int tripId) {
    final channel = WebSocketChannel.connect(Uri.parse('$wsBaseUrl/ws/trips/$tripId'));
    _channel = channel;
    return channel.stream.map((raw) => PositionUpdate.fromJson(jsonDecode(raw as String) as Map<String, dynamic>));
  }

  void close() {
    _channel?.sink.close();
    _channel = null;
  }
}
