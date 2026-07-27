import 'package:flutter/material.dart';

import '../models/account_status.dart';
import '../models/dispatcher_request.dart';
import '../services/api_client.dart';
import '../services/auth_storage.dart';
import '../theme/nocturne_theme.dart';
import 'offered_trips_screen.dart';
import 'vehicle_list_screen.dart';

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
  AccountStatus? _status;
  bool _responding = false;
  bool _leaving = false;

  @override
  void initState() {
    super.initState();
    _reload();
    _loadStatus();
  }

  void _reload() {
    setState(() {
      _requestsFuture = widget.api.listDriverRequests();
    });
  }

  // Only relevant while currently managed - fetches the dispatcher's name to
  // show on the "leave" card below. Non-fatal if it fails: the card still
  // shows without a name.
  Future<void> _loadStatus() async {
    if (widget.api.dispatcherId == null) return;
    try {
      final status = await widget.api.fetchAccountStatus();
      if (!mounted) return;
      setState(() => _status = status);
    } catch (_) {
      // silent
    }
  }

  Future<void> _leaveDispatcher() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Napusti dispečera?'),
        content: const Text(
          'Više nećete videti ponuđene ture od ovog dispečera niti pristup njegovoj floti. '
          'Vaša lična vozila i istorija tura ostaju netaknuti.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text('Otkaži')),
          TextButton(onPressed: () => Navigator.of(context).pop(true), child: const Text('Napusti')),
        ],
      ),
    );
    if (confirmed != true) return;

    setState(() => _leaving = true);
    try {
      await widget.api.leaveDispatcher();
      if (!mounted) return;
      // No longer managed - VehicleListScreen becomes home from here on,
      // same as an independent driver (see entry_router.dart homeScreenFor).
      Navigator.of(context).pushAndRemoveUntil(
        MaterialPageRoute(builder: (_) => VehicleListScreen(api: widget.api)),
        (route) => false,
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Greška: $e')));
    } finally {
      if (mounted) setState(() => _leaving = false);
    }
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

  Widget _currentDispatcherCard() {
    if (widget.api.dispatcherId == null) return const SizedBox.shrink();
    final name = _status?.dispatcherUsername;
    return Card(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            const Icon(Icons.business, color: NocturneColors.accent300),
            const SizedBox(width: 8),
            Expanded(
              child: Text(name != null ? 'Trenutni dispečer: $name' : 'Trenutno ste povezani sa dispečerom'),
            ),
            TextButton(
              onPressed: _leaving ? null : _leaveDispatcher,
              child: _leaving
                  ? const SizedBox(height: 16, width: 16, child: CircularProgressIndicator(strokeWidth: 2))
                  : const Text('Napusti'),
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Zahtevi dispečera')),
      body: Column(
        children: [
          _currentDispatcherCard(),
          Expanded(
            child: FutureBuilder<List<DispatcherRequest>>(
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
          ),
        ],
      ),
    );
  }
}
