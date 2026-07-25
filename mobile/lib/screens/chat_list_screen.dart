import 'package:flutter/material.dart';

import '../models/chat_message.dart';
import '../models/driver.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import 'chat_thread_screen.dart';

/// Mock-up's "Team Chat" panel: GET /api/v1/chats (existing threads) plus a
/// "start new" action backed by GET /api/v1/drivers (every other registered
/// driver - no fleet/team concept in this project).
class ChatListScreen extends StatefulWidget {
  final ApiClient api;
  const ChatListScreen({super.key, required this.api});

  @override
  State<ChatListScreen> createState() => _ChatListScreenState();
}

class _ChatListScreenState extends State<ChatListScreen> {
  late Future<List<ChatConversation>> _chatsFuture;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _chatsFuture = widget.api.listChats();
    });
  }

  Future<void> _openThread(int counterpartId, String? counterpartName) async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => ChatThreadScreen(api: widget.api, counterpartId: counterpartId, counterpartName: counterpartName),
      ),
    );
    _reload();
  }

  Future<void> _startNewChat() async {
    final drivers = await Navigator.of(context).push<Driver>(
      MaterialPageRoute(builder: (_) => _DriverPickerScreen(api: widget.api)),
    );
    if (drivers == null) return;
    if (!mounted) return;
    _openThread(drivers.id, drivers.username);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Poruke')),
      body: FutureBuilder<List<ChatConversation>>(
        future: _chatsFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
          }
          final chats = snapshot.data ?? [];
          if (chats.isEmpty) {
            return const Center(child: Text('Nema razgovora još uvek. Dodirni + za novu poruku.'));
          }
          return ListView.builder(
            itemCount: chats.length,
            itemBuilder: (context, i) {
              final c = chats[i];
              return ListTile(
                leading: CircleAvatar(
                  backgroundColor: NocturneColors.accent800,
                  child: Text('#${c.counterpartId}', style: const TextStyle(fontSize: 11, color: NocturneColors.accent300)),
                ),
                title: Text('Vozač #${c.counterpartId}'),
                subtitle: Text(c.lastMessage, maxLines: 1, overflow: TextOverflow.ellipsis),
                trailing: c.unreadCount > 0
                    ? CircleAvatar(
                        radius: 11,
                        backgroundColor: NocturneColors.accent,
                        child: Text('${c.unreadCount}', style: const TextStyle(fontSize: 11, color: NocturneColors.bg)),
                      )
                    : null,
                onTap: () => _openThread(c.counterpartId, null),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(onPressed: _startNewChat, child: const Icon(Icons.add_comment)),
    );
  }
}

class _DriverPickerScreen extends StatelessWidget {
  final ApiClient api;
  const _DriverPickerScreen({required this.api});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Novi razgovor')),
      body: FutureBuilder<List<Driver>>(
        future: api.listDrivers(),
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
          }
          final drivers = snapshot.data ?? [];
          if (drivers.isEmpty) {
            return const Center(child: Text('Nema drugih registrovanih vozača.'));
          }
          return ListView.builder(
            itemCount: drivers.length,
            itemBuilder: (context, i) {
              final d = drivers[i];
              return ListTile(
                leading: const Icon(Icons.person),
                title: Text(d.username),
                onTap: () => Navigator.of(context).pop(d),
              );
            },
          );
        },
      ),
    );
  }
}
