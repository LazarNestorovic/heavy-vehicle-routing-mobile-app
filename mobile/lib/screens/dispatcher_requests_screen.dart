import 'package:flutter/material.dart';

import '../models/dispatcher_request.dart';
import '../services/api_client.dart';
import '../services/auth_storage.dart';
import '../theme/nocturne_theme.dart';
import 'offered_trips_screen.dart';

/// Driver-side view of incoming dispatcher link requests (see
/// documentations/features/ entry for the dispatcher/driver roles feature).
/// The dispatcher<->driver relationship is established only by approving one
/// of these - never at registration.
class DispatcherRequestsScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherRequestsScreen({super.key, required this.api});

  @override
  State<DispatcherRequestsScreen> createState() => _DispatcherRequestsScreenState();
}

class _DispatcherRequestsScreenState extends State<DispatcherRequestsScreen> {
  late Future<List<DispatcherRequest>> _requestsFuture;
  bool _responding = false;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _requestsFuture = widget.api.listDriverRequests();
    });
  }

  Future<void> _respond(DispatcherRequest req, bool approve) async {
    if (_responding) return;
    setState(() => _responding = true);
    try {
      final dispatcherId = await widget.api.respondDispatcherRequest(req.id, approve);
      if (!mounted) return;

      if (approve && dispatcherId != null) {
        widget.api.dispatcherId = dispatcherId;
        await AuthStorage().saveDispatcherId(dispatcherId);
        if (!mounted) return;
        // Now managed - OfferedTripsScreen becomes the driver's home from here on.
        Navigator.of(context).pushAndRemoveUntil(
          MaterialPageRoute(builder: (_) => OfferedTripsScreen(api: widget.api)),
          (route) => false,
        );
        return;
      }
      _reload();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Greška: $e')));
    } finally {
      if (mounted) setState(() => _responding = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Zahtevi dispečera')),
      body: FutureBuilder<List<DispatcherRequest>>(
        future: _requestsFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
          }
          final requests = snapshot.data ?? [];
          if (requests.isEmpty) {
            return const Center(child: Text('Nema pristiglih zahteva.'));
          }
          return ListView.builder(
            itemCount: requests.length,
            itemBuilder: (context, i) {
              final req = requests[i];
              return Card(
                margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                child: ListTile(
                  leading: const Icon(Icons.business),
                  title: Text(req.dispatcherUsername ?? 'Dispečer #${req.dispatcherId}'),
                  subtitle: const Text('Želi da vas doda u svoju flotu'),
                  trailing: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      IconButton(
                        icon: const Icon(Icons.check, color: NocturneColors.accent),
                        tooltip: 'Odobri',
                        onPressed: _responding ? null : () => _respond(req, true),
                      ),
                      IconButton(
                        icon: const Icon(Icons.close, color: NocturneColors.error),
                        tooltip: 'Odbij',
                        onPressed: _responding ? null : () => _respond(req, false),
                      ),
                    ],
                  ),
                ),
              );
            },
          );
        },
      ),
    );
  }
}
