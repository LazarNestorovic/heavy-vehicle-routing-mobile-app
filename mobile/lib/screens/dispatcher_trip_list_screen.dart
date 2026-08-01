import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import 'dispatcher_live_map_screen.dart';
import 'trip_log_screen.dart';

enum _CompletedSort { dateNewest, dateOldest, distanceLongest, distanceShortest }

/// Dispatcher's version of TripListScreen (see that file's own doc comment
/// for the driver original) - same three-tab shape, but adapted for a
/// non-driving role:
///   - Pokrenute: status "created"/"in_progress" - unlike a driver, a
///     dispatcher can have MANY of these at once (one per driver on the
///     road), so unlike TripListScreen this taps into DispatcherLiveMapScreen
///     (the existing read-only aggregated fleet map) rather than
///     ActiveTripScreen - that screen assumes the CURRENT DEVICE's GPS IS the
///     vehicle's position, which would be wrong for a dispatcher's own phone.
///   - Predstojeće: status "offered"/"accepted".
///   - Završene: status "completed" OR "rejected" (both terminal - a
///     dispatcher especially needs to see rejections, since that's the
///     signal they need to re-offer the trip to someone else), sortable by
///     date/distance like the driver version.
/// Predstojeće/Završene both tap into TripLogScreen (event timeline) - safe
/// for any trip status/role (backend tripAccessible() already permits the
/// assigning dispatcher, not just the driver) and, unlike TripDetailScreen,
/// has no accept/reject/start actions that only make sense for the driver
/// actually on the trip. Replaces the old flat, single-list DispatcherTripsScreen.
class DispatcherTripListScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherTripListScreen({super.key, required this.api});

  @override
  State<DispatcherTripListScreen> createState() => _DispatcherTripListScreenState();
}

class _DispatcherTripListScreenState extends State<DispatcherTripListScreen> {
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
  // status, merge" pattern already used by TripListScreen/ApiClient.findActiveTrip().
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
        title: Text(t.driverUsername ?? 'Vozač #${t.driverId}'),
        subtitle: Text('$dateLabel · ${t.distanceKm.toStringAsFixed(1)} km · ${t.durationMin.toStringAsFixed(0)} min'
            '${t.cargoDescription != null ? " · ${t.cargoDescription}" : ""}'),
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
          title: const Text('Sve ture'),
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
              onTap: (_) => Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => DispatcherLiveMapScreen(api: widget.api)),
              ),
            ),
            _tripList(
              future: _upcomingFuture,
              emptyText: 'Nema predstojećih tura.',
              onTap: (t) => Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => TripLogScreen(api: widget.api, tripId: t.id)),
              ),
            ),
            _completedTab(),
          ],
        ),
      ),
    );
  }
}
