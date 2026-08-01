import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../services/api_client.dart';
import '../services/route_observer.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/active_trip_banner.dart';
import '../widgets/email_verification_banner.dart';
import '../widgets/radial_fab_menu.dart';
import 'chat_list_screen.dart';
import 'dispatcher_requests_screen.dart';
import 'preferences_screen.dart';
import 'profile_screen.dart';
import 'trip_detail_screen.dart';
import 'trip_list_screen.dart';
import 'vehicle_list_screen.dart';

/// Home screen for a MANAGED driver (has a dispatcher) - see
/// documentations/features/ entry for the dispatcher/driver roles feature.
/// They don't plan routes themselves; instead they see trips their
/// dispatcher assigned. Tapping one opens TripDetailScreen (route on map,
/// cargo, vehicle) where they accept/reject an "offered" trip, or start an
/// already-"accepted" one. Profil/Moja vozila/Preference/Zahtevi
/// dispečera/Poruke are reached via a RadialFabMenu fixed on the left edge
/// (non-draggable, hamburger icon) - same pattern as VehicleListScreen, for a
/// consistent navigation layout across the app. "Odjava" lives on
/// ProfileScreen, not duplicated here.
class OfferedTripsScreen extends StatefulWidget {
  final ApiClient api;
  const OfferedTripsScreen({super.key, required this.api});

  @override
  State<OfferedTripsScreen> createState() => _OfferedTripsScreenState();
}

class _OfferedTripsScreenState extends State<OfferedTripsScreen> with RouteAware {
  late Future<List<Trip>> _tripsFuture;
  int _chatUnreadTotal = 0;

  @override
  void initState() {
    super.initState();
    _reload();
    _loadChatUnreadTotal();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final route = ModalRoute.of(context);
    if (route is PageRoute) routeObserver.subscribe(this, route);
  }

  @override
  void dispose() {
    routeObserver.unsubscribe(this);
    super.dispose();
  }

  // Fires when a route pushed on top of this screen (e.g. VehicleListScreen
  // via the "Moja vozila" menu item, whose push isn't awaited/reloaded-after)
  // is popped and this screen becomes visible again - see the identical
  // comment in vehicle_list_screen.dart for why this is needed at all.
  @override
  void didPopNext() => _reload();

  void _reload() {
    setState(() {
      _tripsFuture = Future.wait([
        widget.api.listMyTrips(status: 'offered'),
        widget.api.listMyTrips(status: 'accepted'),
      ]).then((results) => [...results[0], ...results[1]]);
    });
  }

  // Refreshed once on entry (and again whenever the chat list screen is
  // popped back to here) - same scope cut as active_trip_screen.dart's
  // identical method: REST is the source of truth, no live polling.
  Future<void> _loadChatUnreadTotal() async {
    try {
      final chats = await widget.api.listChats();
      if (!mounted) return;
      setState(() => _chatUnreadTotal = chats.fold(0, (sum, c) => sum + c.unreadCount));
    } catch (_) {
      // Non-critical - the badge just stays at its last known value.
    }
  }

  Future<void> _openDetail(Trip trip) async {
    await Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => TripDetailScreen(api: widget.api, trip: trip)),
    );
    _reload();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Ponuđene ture')),
      body: Stack(
        children: [
          Column(
            children: [
              EmailVerificationBanner(api: widget.api),
              ActiveTripBanner(api: widget.api),
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
                        return Center(
                            child:
                                Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
                      }
                      final trips = snapshot.data ?? [];
                      if (trips.isEmpty) {
                        return ListView(
                          children: const [
                            Padding(
                              padding: EdgeInsets.all(32),
                              child:
                                  Center(child: Text('Nema ponuđenih tura trenutno. Povuci na dole za osvežavanje.')),
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
                              title: Text(
                                  '${t.distanceKm.toStringAsFixed(1)} km · ${t.durationMin.toStringAsFixed(0)} min'),
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
          RadialFabMenu(
            initialCorner: FabCorner.bottomLeft,
            draggable: false,
            closedIcon: Icons.menu,
            items: [
              RadialFabMenuItem(
                icon: Icons.person_outline,
                tooltip: 'Profil',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => ProfileScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.local_shipping_outlined,
                tooltip: 'Moja vozila',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => VehicleListScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.tune,
                tooltip: 'Preference',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => PreferencesScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.business,
                tooltip: 'Zahtevi dispečera',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => DispatcherRequestsScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.list_alt,
                tooltip: 'Moje ture',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => TripListScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.chat_bubble_outline,
                tooltip: 'Poruke',
                badgeCount: _chatUnreadTotal,
                onTap: () async {
                  await Navigator.of(context).push(
                    MaterialPageRoute(builder: (_) => ChatListScreen(api: widget.api)),
                  );
                  _loadChatUnreadTotal();
                },
              ),
            ],
          ),
        ],
      ),
    );
  }
}
