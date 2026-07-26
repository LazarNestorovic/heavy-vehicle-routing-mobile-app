import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../config.dart';
import '../models/chat_message.dart';

/// Wraps GET /ws/chats/{counterpartId} (see backend/internal/ws/chat_gateway.go)
/// as a Dart Stream of live-delivered messages. Sending still goes through
/// ApiClient.sendChatMessage (REST) - this socket is receive-only, same split
/// as the backend gateway. Token travels as a query parameter, not a header -
/// same reason as TripSocket (browsers' WebSocket API can't set custom headers).
///
/// Reconnects automatically (with backoff) on drop, same as TripSocket - a
/// chat thread open in the background shouldn't silently stop delivering
/// messages after a brief network hiccup.
class ChatSocket {
  static const _retryDelays = [Duration(seconds: 1), Duration(seconds: 2), Duration(seconds: 5), Duration(seconds: 10)];

  WebSocketChannel? _channel;
  StreamController<ChatMessage>? _controller;
  int? _counterpartId;
  String? _token;
  bool _closed = false;
  int _retryCount = 0;

  Stream<ChatMessage> connect(int counterpartId, String token) {
    _counterpartId = counterpartId;
    _token = token;
    _closed = false;
    _controller = StreamController<ChatMessage>.broadcast(onCancel: close);
    _open();
    return _controller!.stream;
  }

  void _open() {
    final uri = Uri.parse('$wsBaseUrl/ws/chats/$_counterpartId').replace(queryParameters: {'token': _token});
    final channel = WebSocketChannel.connect(uri);
    _channel = channel;
    channel.stream.listen(
      (raw) {
        _retryCount = 0;
        _controller?.add(ChatMessage.fromJson(jsonDecode(raw as String) as Map<String, dynamic>));
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
