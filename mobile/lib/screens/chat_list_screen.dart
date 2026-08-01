import 'dart:async';

import 'package:flutter/material.dart';

import '../models/chat_message.dart';
import '../models/driver.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import 'chat_thread_screen.dart';

/// Mock-up's "Team Chat" panel: GET /api/v1/chats (existing threads) plus a
/// "start new" action backed by GET /api/v1/drivers (every other registered
/// account, driver or dispatcher - no fleet/team concept restricting who can
/// message whom). Reachable from both roles' RadialFabMenu - see
/// documentations/features/ entry for the chat access + contact search +
/// pinning feature.
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
    final contact = await Navigator.of(context).push<Driver>(
      MaterialPageRoute(builder: (_) => _ContactPickerScreen(api: widget.api)),
    );
    if (contact == null) return;
    if (!mounted) return;
    _openThread(contact.id, contact.username);
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
                  child: Text(c.counterpartUsername.substring(0, 1).toUpperCase(),
                      style: const TextStyle(fontSize: 14, color: NocturneColors.accent300)),
                ),
                title: Text(c.counterpartUsername),
                subtitle: Text(c.lastMessage, maxLines: 1, overflow: TextOverflow.ellipsis),
                trailing: c.unreadCount > 0
                    ? CircleAvatar(
                        radius: 11,
                        backgroundColor: NocturneColors.accent,
                        child: Text('${c.unreadCount}', style: const TextStyle(fontSize: 11, color: NocturneColors.bg)),
                      )
                    : null,
                onTap: () => _openThread(c.counterpartId, c.counterpartUsername),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(onPressed: _startNewChat, child: const Icon(Icons.add_comment)),
    );
  }
}

/// "Start a new chat" contact picker - GET /api/v1/drivers, searchable by
/// name or email (same debounced-TextField pattern as
/// dispatcher_available_drivers_screen.dart). The caller's own dispatcher
/// (if a managed driver) or own managed drivers (if a dispatcher) are pinned
/// to the top of the list - see documentations/features/ entry. Pinning
/// happens client-side: [ApiClient.dispatcherId] and
/// [ApiClient.listManagedDrivers] are already available/cheap, so no new
/// backend endpoint is needed just to know who to pin.
class _ContactPickerScreen extends StatefulWidget {
  final ApiClient api;
  const _ContactPickerScreen({required this.api});

  @override
  State<_ContactPickerScreen> createState() => _ContactPickerScreenState();
}

class _ContactPickerScreenState extends State<_ContactPickerScreen> {
  late Future<(List<Driver>, Set<int>)> _dataFuture;
  final _searchCtrl = TextEditingController();
  Timer? _debounce;

  // Fetched once (not on every search reload, unlike the contact list
  // itself) - the caller's own dispatcher/managed-drivers relationship
  // doesn't change while this screen is open.
  late final Future<Set<int>> _pinnedIdsFuture = _loadPinnedIds();

  @override
  void initState() {
    super.initState();
    _reload();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<Set<int>> _loadPinnedIds() async {
    if (widget.api.role == 'dispatcher') {
      final managed = await widget.api.listManagedDrivers();
      return managed.map((d) => d.id).toSet();
    }
    final dispatcherId = widget.api.dispatcherId;
    return dispatcherId == null ? {} : {dispatcherId};
  }

  void _reload() {
    setState(() {
      _dataFuture = Future.wait([
        widget.api.listDrivers(query: _searchCtrl.text.trim()),
        _pinnedIdsFuture,
      ]).then((results) => (results[0] as List<Driver>, results[1] as Set<int>));
    });
  }

  // setState fires immediately (so the clear icon shows/hides right away);
  // the actual reload is debounced so a request isn't fired on every keystroke.
  void _onSearchChanged(String _) {
    setState(() {});
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), _reload);
  }

  // Pinned contacts first (dispatcher's managed drivers, or a managed
  // driver's own dispatcher) - order within each group stays whatever the
  // backend returned (alphabetical by username), since order among the
  // pinned entries themselves doesn't matter.
  List<Driver> _sorted(List<Driver> contacts, Set<int> pinnedIds) {
    final pinned = contacts.where((d) => pinnedIds.contains(d.id)).toList();
    final rest = contacts.where((d) => !pinnedIds.contains(d.id)).toList();
    return [...pinned, ...rest];
  }

  @override
  Widget build(BuildContext context) {
    final pinnedLabel = widget.api.role == 'dispatcher' ? 'Vaš vozač' : 'Dispečer';
    return Scaffold(
      appBar: AppBar(title: const Text('Novi razgovor')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: TextField(
              controller: _searchCtrl,
              onChanged: _onSearchChanged,
              decoration: InputDecoration(
                hintText: 'Pretraga po imenu ili mejlu...',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _searchCtrl.text.isEmpty
                    ? null
                    : IconButton(
                        icon: const Icon(Icons.clear),
                        onPressed: () {
                          _searchCtrl.clear();
                          _reload();
                        },
                      ),
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                isDense: true,
              ),
            ),
          ),
          Expanded(
            child: FutureBuilder<(List<Driver>, Set<int>)>(
              future: _dataFuture,
              builder: (context, snapshot) {
                if (snapshot.connectionState == ConnectionState.waiting) {
                  return const Center(child: CircularProgressIndicator());
                }
                if (snapshot.hasError) {
                  return Center(
                      child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
                }
                final (contacts, pinnedIds) = snapshot.data!;
                if (contacts.isEmpty) {
                  final noMatches = _searchCtrl.text.trim().isNotEmpty;
                  return Center(
                    child: Text(noMatches ? 'Nema rezultata za ovu pretragu.' : 'Nema drugih registrovanih korisnika.'),
                  );
                }
                final sorted = _sorted(contacts, pinnedIds);
                return ListView.builder(
                  itemCount: sorted.length,
                  itemBuilder: (context, i) {
                    final d = sorted[i];
                    final isPinned = pinnedIds.contains(d.id);
                    return ListTile(
                      leading: Icon(isPinned ? Icons.star : Icons.person_outline,
                          color: isPinned ? NocturneColors.accent300 : null),
                      title: Text(d.username),
                      subtitle: d.email == null ? null : Text(d.email!),
                      trailing: isPinned
                          ? Text(pinnedLabel, style: const TextStyle(color: NocturneColors.accent300))
                          : null,
                      onTap: () => Navigator.of(context).pop(d),
                    );
                  },
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}
