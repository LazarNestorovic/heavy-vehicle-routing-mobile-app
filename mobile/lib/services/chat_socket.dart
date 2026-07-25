import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config.dart';
import '../models/chat_message.dart';

/// Wraps GET /ws/chats/{counterpartId} (see backend/internal/ws/chat_gateway.go)
/// as a Dart Stream of live-delivered messages. Sending still goes through
/// ApiClient.sendChatMessage (REST) - this socket is receive-only, same split
/// as the backend gateway. Token travels as a query parameter, not a header -
/// same reason as TripSocket (browsers' WebSocket API can't set custom headers).
class ChatSocket {
  WebSocketChannel? _channel;

  Stream<ChatMessage> connect(int counterpartId, String token) {
    final uri = Uri.parse('$wsBaseUrl/ws/chats/$counterpartId').replace(queryParameters: {'token': token});
    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    return channel.stream.map((raw) => ChatMessage.fromJson(jsonDecode(raw as String) as Map<String, dynamic>));
  }

  void close() {
    _channel?.sink.close();
    _channel = null;
  }
}
