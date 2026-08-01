import 'package:flutter/material.dart';

import '../models/trip_event.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Mock-up's "Trip Log" panel: GET /api/v1/trips/{id}/events timeline
/// (departed / rest_stop_suggested / arrived - see backend/internal/worker
/// and backend/internal/ws/gateway.go for where each is written).
class TripLogScreen extends StatefulWidget {
  final ApiClient api;
  final int tripId;
  const TripLogScreen({super.key, required this.api, required this.tripId});

  @override
  State<TripLogScreen> createState() => _TripLogScreenState();
}

class _TripLogScreenState extends State<TripLogScreen> {
  late Future<List<TripEvent>> _eventsFuture;

  @override
  void initState() {
    super.initState();
    _eventsFuture = widget.api.listTripEvents(widget.tripId);
  }

  IconData _iconFor(String eventType) {
    switch (eventType) {
      case 'departed':
        return Icons.trip_origin;
      case 'rest_stop_suggested':
        return Icons.bedtime;
      case 'arrived':
        return Icons.flag;
      case 'edited':
        return Icons.edit_outlined;
      default:
        return Icons.circle;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Dnevnik putovanja')),
      body: FutureBuilder<List<TripEvent>>(
        future: _eventsFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
          }
          final events = snapshot.data ?? [];
          if (events.isEmpty) {
            return const Center(child: Text('Nema zabeleženih dogadjaja.'));
          }
          return ListView.builder(
            padding: const EdgeInsets.all(8),
            itemCount: events.length,
            itemBuilder: (context, i) {
              final e = events[i];
              return ListTile(
                leading: Icon(_iconFor(e.eventType), color: NocturneColors.accent),
                title: Text(e.description),
                subtitle: Text(e.occurredAt.toLocal().toString()),
              );
            },
          );
        },
      ),
    );
  }
}
