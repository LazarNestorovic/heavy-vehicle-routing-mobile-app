import 'dart:async';

import 'package:flutter/material.dart';

import '../models/dispatcher_request.dart';
import '../models/driver.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Dispatcher's "send a link request" screen - lists every unmanaged driver
/// (GET /api/v1/dispatcher/available-drivers), searchable by name or email,
/// plus the status of requests already sent (GET /api/v1/dispatcher/requests).
/// See documentations/features/ entry for the dispatcher/driver roles feature
/// and the search-by-name/email addition.
class DispatcherAvailableDriversScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherAvailableDriversScreen({super.key, required this.api});

  @override
  State<DispatcherAvailableDriversScreen> createState() => _DispatcherAvailableDriversScreenState();
}

class _DispatcherAvailableDriversScreenState extends State<DispatcherAvailableDriversScreen> {
  late Future<(List<Driver>, List<DispatcherRequest>)> _dataFuture;
  bool _sending = false;
  final _searchCtrl = TextEditingController();
  Timer? _debounce;

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

  void _reload() {
    setState(() {
      _dataFuture = Future.wait([
        widget.api.listAvailableDrivers(query: _searchCtrl.text.trim()),
        widget.api.listSentDispatcherRequests(),
      ]).then((results) => (results[0] as List<Driver>, results[1] as List<DispatcherRequest>));
    });
  }

  // setState fires immediately (so the clear icon shows/hides right away);
  // the actual reload is debounced so a request isn't fired on every keystroke.
  void _onSearchChanged(String _) {
    setState(() {});
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), _reload);
  }

  Future<void> _sendRequest(Driver driver) async {
    if (_sending) return;
    setState(() => _sending = true);
    try {
      await widget.api.sendDispatcherRequest(driver.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Zahtev poslat vozaču ${driver.username}.')));
      _reload();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Greška: $e')));
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Dostupni vozači')),
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
            child: RefreshIndicator(
              onRefresh: () async => _reload(),
              child: FutureBuilder<(List<Driver>, List<DispatcherRequest>)>(
                future: _dataFuture,
                builder: (context, snapshot) {
                  if (snapshot.connectionState == ConnectionState.waiting) {
                    return const Center(child: CircularProgressIndicator());
                  }
                  if (snapshot.hasError) {
                    return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
                  }
                  final (drivers, sentRequests) = snapshot.data!;
                  if (drivers.isEmpty) {
                    final noMatches = _searchCtrl.text.trim().isNotEmpty;
                    return ListView(
                      children: [
                        Padding(
                          padding: const EdgeInsets.all(32),
                          child: Center(
                            child: Text(noMatches
                                ? 'Nema vozača koji odgovaraju pretrazi.'
                                : 'Nema dostupnih (neupravljanih) vozača trenutno.'),
                          ),
                        ),
                      ],
                    );
                  }
                  return ListView.builder(
                    itemCount: drivers.length,
                    itemBuilder: (context, i) {
                      final d = drivers[i];
                      final pending = sentRequests.any((r) => r.driverId == d.id && r.status == 'pending');
                      return ListTile(
                        leading: const Icon(Icons.person_outline),
                        title: Text(d.username),
                        subtitle: d.email == null ? null : Text(d.email!),
                        trailing: pending
                            ? const Text('Poslato', style: TextStyle(color: NocturneColors.accent300))
                            : OutlinedButton(
                                onPressed: _sending ? null : () => _sendRequest(d),
                                child: const Text('Pošalji zahtev'),
                              ),
                      );
                    },
                  );
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}
