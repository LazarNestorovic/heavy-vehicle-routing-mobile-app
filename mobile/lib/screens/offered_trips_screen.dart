import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/email_verification_banner.dart';
import 'dispatcher_requests_screen.dart';
import 'entry_router.dart';
import 'login_screen.dart';
import 'preferences_screen.dart';
import 'profile_screen.dart';
import 'trip_detail_screen.dart';
import 'vehicle_list_screen.dart';

/// Home screen for a MANAGED driver (has a dispatcher) - see
/// documentations/features/ entry for the dispatcher/driver roles feature.
/// They don't plan routes themselves; instead they see trips their
/// dispatcher assigned. Tapping one opens TripDetailScreen (route on map,
/// cargo, vehicle) where they accept/reject an "offered" trip, or start an
/// already-"accepted" one.
class OfferedTripsScreen extends StatefulWidget {
  final ApiClient api;
  const OfferedTripsScreen({super.key, required this.api});

  @override
  State<OfferedTripsScreen> createState() => _OfferedTripsScreenState();
}

class _OfferedTripsScreenState extends State<OfferedTripsScreen> {
  late Future<List<Trip>> _tripsFuture;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _tripsFuture = Future.wait([
        widget.api.listMyTrips(status: 'offered'),
        widget.api.listMyTrips(status: 'accepted'),
      ]).then((results) => [...results[0], ...results[1]]);
    });
  }

  Future<void> _openDetail(Trip trip) async {
    await Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => TripDetailScreen(api: widget.api, trip: trip)),
    );
    _reload();
  }

  Future<void> _logout() async {
    await clearSession(widget.api);
    if (!mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => LoginScreen(api: widget.api)),
      (route) => false,
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Ponuđene ture'),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_outline),
            tooltip: 'Profil',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => ProfileScreen(api: widget.api)),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.local_shipping_outlined),
            tooltip: 'Moja vozila',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => VehicleListScreen(api: widget.api)),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.tune),
            tooltip: 'Preference',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => PreferencesScreen(api: widget.api)),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.business),
            tooltip: 'Zahtevi dispečera',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => DispatcherRequestsScreen(api: widget.api)),
            ),
          ),
          IconButton(icon: const Icon(Icons.logout), tooltip: 'Odjava', onPressed: _logout),
        ],
      ),
      body: Column(
        children: [
          EmailVerificationBanner(api: widget.api),
          Expanded(
            child: RefreshIndicator(
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
                          child: Center(child: Text('Nema ponuđenih tura trenutno. Povuci na dole za osvežavanje.')),
                        ),
                      ],
                    );
                  }
                  return ListView.builder(
                    itemCount: trips.length,
                    itemBuilder: (context, i) {
                      final t = trips[i];
                      final isAccepted = t.status == 'accepted';
                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                        child: ListTile(
                          leading: Icon(isAccepted ? Icons.check_circle_outline : Icons.local_shipping),
                          title: Text('${t.distanceKm.toStringAsFixed(1)} km · ${t.durationMin.toStringAsFixed(0)} min'),
                          subtitle: Text(isAccepted
                              ? '${t.cargoDescription ?? "Bez opisa tovara"} · prihvaćena, spremna za polazak'
                              : t.cargoDescription ?? 'Bez opisa tovara'),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => _openDetail(t),
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
