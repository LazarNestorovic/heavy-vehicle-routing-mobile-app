import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import 'active_trip_screen.dart';
import 'trip_detail_screen.dart';
import 'trip_log_screen.dart';

enum _CompletedSort { dateNewest, dateOldest, distanceLongest, distanceShortest }

/// "Moje ture" - a driver's trips grouped into the standard three tabs for
/// this kind of screen (active/upcoming/history) - addresses a real gap
/// neither driver type previously had: no way to see completed trip history,
/// or an active trip alongside upcoming ones, all in one place.
///   - Pokrenute: status "created"/"in_progress" - at most one (backend
///     enforces a single active trip, see HasActiveTrip) - taps straight
///     into ActiveTripScreen's live map.
///   - Predstojeće: status "offered"/"accepted" - only ever non-empty for a
///     MANAGED driver (a self-service trip skips straight to "created", no
///     offer/accept step) - taps into TripDetailScreen, same destination as
///     OfferedTripsScreen's own list.
///   - Završene: status "completed" OR "rejected" - both are terminal, nothing
///     more will happen to either (a rejected offer previously had NO home in
///     any list anywhere in the app - folded in here rather than adding a
///     4th tab for a rare edge case). Sortable by date or distance (a small,
///     standard convenience for a history list, entirely client-side - the
///     list is thesis-scale, not worth a backend sort parameter) - taps into
///     TripLogScreen (the trip's departed/rest-stop/arrived/rerouted
///     timeline).
class TripListScreen extends StatefulWidget {
  final ApiClient api;
  const TripListScreen({super.key, required this.api});

  @override
  State<TripListScreen> createState() => _TripListScreenState();
}

class _TripListScreenState extends State<TripListScreen> {
  late Future<List<Trip>> _activeFuture;
  late Future<List<Trip>> _upcomingFuture;
  late Future<List<Trip>> _completedFuture;
  _CompletedSort _completedSort = _CompletedSort.dateNewest;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _activeFuture = _fetchStatuses(['created', 'in_progress']);
      _upcomingFuture = _fetchStatuses(['offered', 'accepted']);
      _completedFuture = _fetchStatuses(['completed', 'rejected']);
    });
  }

  // GET /trips only filters by a single status at a time - same "fetch each
  // status, merge" pattern already used by ApiClient.findActiveTrip().
  Future<List<Trip>> _fetchStatuses(List<String> statuses) async {
    final results = await Future.wait(statuses.map((s) => widget.api.listMyTrips(status: s)));
    return results.expand((trips) => trips).toList();
  }

  List<Trip> _sortCompleted(List<Trip> trips) {
    final sorted = [...trips];
    switch (_completedSort) {
      case _CompletedSort.dateNewest:
        sorted.sort((a, b) => b.createdAt.compareTo(a.createdAt));
      case _CompletedSort.dateOldest:
        sorted.sort((a, b) => a.createdAt.compareTo(b.createdAt));
      case _CompletedSort.distanceLongest:
        sorted.sort((a, b) => b.distanceKm.compareTo(a.distanceKm));
      case _CompletedSort.distanceShortest:
        sorted.sort((a, b) => a.distanceKm.compareTo(b.distanceKm));
    }
    return sorted;
  }

  IconData _statusIcon(String status) {
    switch (status) {
      case 'created':
      case 'in_progress':
        return Icons.local_shipping;
      case 'offered':
        return Icons.mail_outline;
      case 'accepted':
        return Icons.check_circle_outline;
      case 'completed':
        return Icons.flag_outlined;
      case 'rejected':
        return Icons.cancel_outlined;
      default:
        return Icons.circle;
    }
  }

  Color _statusColor(String status) {
    switch (status) {
      case 'created':
      case 'in_progress':
        return Colors.green;
      case 'offered':
        return NocturneColors.accent300;
      case 'accepted':
        return NocturneColors.accent;
      case 'rejected':
        return NocturneColors.error;
      default:
        return NocturneColors.text;
    }
  }

  Widget _tripTile(Trip t, {required VoidCallback onTap}) {
    final date = t.createdAt.toLocal();
    final dateLabel = '${date.day.toString().padLeft(2, '0')}.${date.month.toString().padLeft(2, '0')}.${date.year}.';
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: ListTile(
        leading: Icon(_statusIcon(t.status), color: _statusColor(t.status)),
        title: Text('${t.distanceKm.toStringAsFixed(1)} km · ${t.durationMin.toStringAsFixed(0)} min'),
        subtitle: Text('$dateLabel${t.cargoDescription != null ? " · ${t.cargoDescription}" : ""}'),
        trailing: const Icon(Icons.chevron_right),
        onTap: onTap,
      ),
    );
  }

  Widget _tripList({
    required Future<List<Trip>> future,
    required String emptyText,
    required void Function(Trip) onTap,
  }) {
    return FutureBuilder<List<Trip>>(
      future: future,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Center(child: CircularProgressIndicator());
        }
        if (snapshot.hasError) {
          return Center(child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
        }
        final trips = snapshot.data ?? [];
        return RefreshIndicator(
          onRefresh: () async => _reload(),
          child: trips.isEmpty
              ? ListView(
                  children: [Padding(padding: const EdgeInsets.all(32), child: Center(child: Text(emptyText)))],
                )
              : ListView.builder(
                  itemCount: trips.length,
                  itemBuilder: (context, i) => _tripTile(trips[i], onTap: () => onTap(trips[i])),
                ),
        );
      },
    );
  }

  Widget _completedTab() {
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
          child: Row(
            children: [
              const Spacer(),
              DropdownButton<_CompletedSort>(
                value: _completedSort,
                underline: const SizedBox.shrink(),
                items: const [
                  DropdownMenuItem(value: _CompletedSort.dateNewest, child: Text('Datum (najnovije)')),
                  DropdownMenuItem(value: _CompletedSort.dateOldest, child: Text('Datum (najstarije)')),
                  DropdownMenuItem(value: _CompletedSort.distanceLongest, child: Text('Rastojanje (najduže)')),
                  DropdownMenuItem(value: _CompletedSort.distanceShortest, child: Text('Rastojanje (najkraće)')),
                ],
                onChanged: (v) => setState(() => _completedSort = v ?? _completedSort),
              ),
            ],
          ),
        ),
        Expanded(
          child: FutureBuilder<List<Trip>>(
            future: _completedFuture,
            builder: (context, snapshot) {
              if (snapshot.connectionState == ConnectionState.waiting) {
                return const Center(child: CircularProgressIndicator());
              }
              if (snapshot.hasError) {
                return Center(
                    child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
              }
              final trips = _sortCompleted(snapshot.data ?? []);
              return RefreshIndicator(
                onRefresh: () async => _reload(),
                child: trips.isEmpty
                    ? ListView(
                        children: const [
                          Padding(padding: EdgeInsets.all(32), child: Center(child: Text('Nema završenih tura.'))),
                        ],
                      )
                    : ListView.builder(
                        itemCount: trips.length,
                        itemBuilder: (context, i) => _tripTile(
                          trips[i],
                          onTap: () => Navigator.of(context).push(
                            MaterialPageRoute(builder: (_) => TripLogScreen(api: widget.api, tripId: trips[i].id)),
                          ),
                        ),
                      ),
              );
            },
          ),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('Moje ture'),
          bottom: const TabBar(
            tabs: [
              Tab(text: 'Pokrenute'),
              Tab(text: 'Predstojeće'),
              Tab(text: 'Završene'),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            _tripList(
              future: _activeFuture,
              emptyText: 'Nema pokrenutih tura.',
              onTap: (t) => Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => ActiveTripScreen(api: widget.api, trip: t, vehicleId: t.vehicleId)),
              ),
            ),
            _tripList(
              future: _upcomingFuture,
              emptyText: 'Nema predstojećih tura.',
              onTap: (t) async {
                await Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => TripDetailScreen(api: widget.api, trip: t)),
                );
                _reload();
              },
            ),
            _completedTab(),
          ],
        ),
      ),
    );
  }
}
