import 'package:flutter/material.dart';

import '../models/dispatcher_request.dart';
import '../models/driver.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Dispatcher's "send a link request" screen - lists every unmanaged driver
/// (GET /api/v1/dispatcher/available-drivers) plus the status of requests
/// already sent (GET /api/v1/dispatcher/requests). See documentations/
/// features/ entry for the dispatcher/driver roles feature.
class DispatcherAvailableDriversScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherAvailableDriversScreen({super.key, required this.api});

  @override
  State<DispatcherAvailableDriversScreen> createState() => _DispatcherAvailableDriversScreenState();
}

class _DispatcherAvailableDriversScreenState extends State<DispatcherAvailableDriversScreen> {
  late Future<(List<Driver>, List<DispatcherRequest>)> _dataFuture;
  bool _sending = false;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _dataFuture = Future.wait([
        widget.api.listAvailableDrivers(),
        widget.api.listSentDispatcherRequests(),
      ]).then((results) => (results[0] as List<Driver>, results[1] as List<DispatcherRequest>));
    });
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
      body: RefreshIndicator(
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
              return ListView(
                children: const [
                  Padding(
                    padding: EdgeInsets.all(32),
                    child: Center(child: Text('Nema dostupnih (neupravljanih) vozača trenutno.')),
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
    );
  }
}
