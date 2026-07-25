import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Dispatcher's view of every trip they've assigned - addresses a real gap
/// found in live testing: after sending a trip, the dispatcher had no way to
/// see whether it was still pending, accepted, or already under way. See
/// documentations/features/ entry for the dispatcher/driver roles feature.
class DispatcherTripsScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherTripsScreen({super.key, required this.api});

  @override
  State<DispatcherTripsScreen> createState() => _DispatcherTripsScreenState();
}

class _DispatcherTripsScreenState extends State<DispatcherTripsScreen> {
  late Future<List<Trip>> _tripsFuture;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _tripsFuture = widget.api.listMyTrips();
    });
  }

  static const _statusLabels = {
    'offered': 'Ponuđena',
    'accepted': 'Prihvaćena',
    'created': 'Pokrenuta',
    'in_progress': 'Pokrenuta',
    'rejected': 'Odbijena',
  };

  Color _statusColor(String status) {
    switch (status) {
      case 'offered':
        return NocturneColors.accent300;
      case 'accepted':
        return NocturneColors.accent;
      case 'created':
      case 'in_progress':
        return Colors.green;
      case 'rejected':
        return NocturneColors.error;
      default:
        return NocturneColors.text;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Sve ture')),
      body: RefreshIndicator(
        onRefresh: () async => _reload(),
        child: FutureBuilder<List<Trip>>(
          future: _tripsFuture,
          builder: (context, snapshot) {
            if (snapshot.connectionState == ConnectionState.waiting) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snapshot.hasError) {
              return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
            }
            final trips = snapshot.data ?? [];
            if (trips.isEmpty) {
              return ListView(
                children: const [
                  Padding(
                    padding: EdgeInsets.all(32),
                    child: Center(child: Text('Nema dodeljenih tura još uvek.')),
                  ),
                ],
              );
            }
            return ListView.builder(
              itemCount: trips.length,
              itemBuilder: (context, i) {
                final t = trips[i];
                return Card(
                  margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                  child: ListTile(
                    leading: const Icon(Icons.person_outline),
                    title: Text(t.driverUsername ?? 'Vozač #${t.driverId}'),
                    subtitle: Text('${t.distanceKm.toStringAsFixed(1)} km · ${t.durationMin.toStringAsFixed(0)} min'
                        '${t.cargoDescription != null ? " · ${t.cargoDescription}" : ""}'),
                    trailing: Text(
                      _statusLabels[t.status] ?? t.status,
                      style: TextStyle(color: _statusColor(t.status), fontWeight: FontWeight.w600),
                    ),
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
