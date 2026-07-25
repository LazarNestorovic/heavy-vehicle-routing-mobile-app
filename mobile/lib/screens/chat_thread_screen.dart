import 'package:flutter/material.dart';

import '../models/chat_message.dart';
import '../services/api_client.dart';
import '../services/chat_socket.dart';
import '../theme/nocturne_theme.dart';

/// Mock-up's chat thread view: history via GET (marks read as a side effect,
/// see backend/internal/httpapi/chat.go handleGetChatMessages), live delivery
/// via ChatSocket (GET /ws/chats/{counterpartId}), sending via POST. The
/// socket is receive-only - sent messages are appended locally from the POST
/// response, not waited for over the socket.
class ChatThreadScreen extends StatefulWidget {
  final ApiClient api;
  final int counterpartId;
  final String? counterpartName;

  const ChatThreadScreen({super.key, required this.api, required this.counterpartId, this.counterpartName});

  @override
  State<ChatThreadScreen> createState() => _ChatThreadScreenState();
}

class _ChatThreadScreenState extends State<ChatThreadScreen> {
  final _socket = ChatSocket();
  final _textCtrl = TextEditingController();
  final _scrollCtrl = ScrollController();

  List<ChatMessage> _messages = [];
  bool _loading = true;
  bool _sending = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
    _listen();
  }

  Future<void> _load() async {
    try {
      final messages = await widget.api.getChatMessages(widget.counterpartId);
      if (!mounted) return;
      setState(() {
        _messages = messages;
        _loading = false;
      });
      _scrollToEnd();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = 'Greška: $e';
        _loading = false;
      });
    }
  }

  void _listen() {
    _socket.connect(widget.counterpartId, widget.api.token ?? '').listen(
      (message) {
        if (!mounted) return;
        _addMessage(message);
      },
      onError: (_) {}, // live delivery is best-effort - REST history remains the source of truth
    );
  }

  // The chat.events routing key doesn't distinguish direction, so the sender's
  // own open WS connection also receives the message they just sent - without
  // this id-based dedup it would show up twice (once from the POST response
  // appended in _send, once pushed back over the socket).
  void _addMessage(ChatMessage message) {
    if (_messages.any((m) => m.id == message.id)) return;
    setState(() => _messages = [..._messages, message]);
    _scrollToEnd();
  }

  void _scrollToEnd() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollCtrl.hasClients) {
        _scrollCtrl.jumpTo(_scrollCtrl.position.maxScrollExtent);
      }
    });
  }

  Future<void> _send() async {
    final body = _textCtrl.text.trim();
    if (body.isEmpty || _sending) return;

    setState(() => _sending = true);
    try {
      final sent = await widget.api.sendChatMessage(widget.counterpartId, body);
      if (!mounted) return;
      _addMessage(sent);
      _textCtrl.clear();
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Greška: $e');
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  void dispose() {
    _socket.close();
    _textCtrl.dispose();
    _scrollCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.counterpartName ?? 'Vozač #${widget.counterpartId}')),
      body: Column(
        children: [
          if (_error != null)
            Padding(
              padding: const EdgeInsets.all(8),
              child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
            ),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : ListView.builder(
                    controller: _scrollCtrl,
                    padding: const EdgeInsets.all(12),
                    itemCount: _messages.length,
                    itemBuilder: (context, i) {
                      final m = _messages[i];
                      final isMine = m.fromDriverId == widget.api.driverId;
                      return Align(
                        alignment: isMine ? Alignment.centerRight : Alignment.centerLeft,
                        child: Container(
                          margin: const EdgeInsets.symmetric(vertical: 4),
                          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                          constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.75),
                          decoration: BoxDecoration(
                            color: isMine ? NocturneColors.accent800 : NocturneColors.surface,
                            borderRadius: BorderRadius.circular(NocturneRadii.md),
                          ),
                          child: Text(m.body),
                        ),
                      );
                    },
                  ),
          ),
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _textCtrl,
                      decoration: const InputDecoration(hintText: 'Poruka...'),
                      onSubmitted: (_) => _send(),
                    ),
                  ),
                  const SizedBox(width: 8),
                  IconButton(
                    onPressed: _sending ? null : _send,
                    icon: const Icon(Icons.send),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
