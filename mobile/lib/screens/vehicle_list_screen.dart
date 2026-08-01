import 'dart:async';

import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/route_observer.dart';
import '../widgets/active_trip_banner.dart';
import '../widgets/email_verification_banner.dart';
import '../widgets/radial_fab_menu.dart';
import 'active_trip_screen.dart';
import 'chat_list_screen.dart';
import 'dispatcher_requests_screen.dart';
import 'preferences_screen.dart';
import 'profile_screen.dart';
import 'route_request_screen.dart';
import 'trip_list_screen.dart';
import 'vehicle_profile_screen.dart';

/// Hub screen after login for a SELF-SERVICE driver (SPECIFIKACIJA.md 3.9 +
/// Faza 2/6 of the driver-preference plan): lists the driver's own vehicles
/// (1:N, see documentations/features/2026-07-21-driver-preference-scoring.md),
/// pick one to start planning a route, or add a new one. Also reused, as a
/// SECONDARY destination, by:
///   - a MANAGED driver (has a dispatcher), from the "Moja vozila" item on
///     OfferedTripsScreen's own menu - for vehicle CRUD only, no route
///     planning (their dispatcher assigns trips).
///   - a DISPATCHER, from the "Vozila" item on DispatcherHomeScreen's own
///     menu - lists/manages their fleet (GET /vehicles already returns the
///     fleet for a dispatcher account, see backend handleListVehicles).
/// The RadialFabMenu below is shown ONLY for the self-service case (this
/// screen IS their home there); both secondary cases already have an
/// identical Profil/Preference/... menu on the screen they came from, so a
/// second one here would be a confusing duplicate. "Odjava" lives on
/// ProfileScreen, not duplicated here; "Dodaj vozilo" stays a plain,
/// always-visible FAB for all three cases since it's this screen's primary
/// action, not a secondary one.
class VehicleListScreen extends StatefulWidget {
  final ApiClient api;
  const VehicleListScreen({super.key, required this.api});

  @override
  State<VehicleListScreen> createState() => _VehicleListScreenState();
}

class _VehicleListScreenState extends State<VehicleListScreen> with RouteAware {
  late Future<List<VehicleProfile>> _vehiclesFuture;

  // Lets tapping the vehicle that's ON the active trip jump straight to
  // ActiveTripScreen instead of RouteRequestScreen (where it would just hit
  // the "already have an active trip" block - see route_request_screen.dart).
  // Separate from ActiveTripBanner's own internal fetch (that widget stays
  // self-contained for OfferedTripsScreen's sake) - a second lightweight GET
  // is an acceptable cost for keeping both independent.
  Trip? _activeTrip;
  int _chatUnreadTotal = 0;

  @override
  void initState() {
    super.initState();
    _reload();
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

  // Fires when a route pushed on top of this screen (RouteRequestScreen, or
  // ActiveTripScreen reached via a vehicle tap) is popped and this screen
  // becomes visible again - Flutter does NOT call initState() again for it,
  // so without this the vehicle list AND the active-trip tap-routing (see
  // onTap below) both kept showing stale state until the whole app restarted.
  @override
  void didPopNext() => _reload();

  void _reload() {
    setState(() {
      _vehiclesFuture = widget.api.listVehicles();
    });
    unawaited(_loadActiveTrip());
    if (!_isDispatcher && widget.api.dispatcherId == null) _loadChatUnreadTotal();
  }

  // Refreshed on entry and whenever a pushed route (e.g. the chat list
  // screen) is popped back to here - same scope cut as active_trip_screen.
  // dart's identical method: REST is the source of truth, no live polling.
  Future<void> _loadChatUnreadTotal() async {
    try {
      final chats = await widget.api.listChats();
      if (!mounted) return;
      setState(() => _chatUnreadTotal = chats.fold(0, (sum, c) => sum + c.unreadCount));
    } catch (_) {
      // Non-critical - the badge just stays at its last known value.
    }
  }

  Future<void> _loadActiveTrip() async {
    // A dispatcher account has no trips of their own to resume - see
    // ActiveTripBanner's identical guard for why.
    if (_isDispatcher) return;
    try {
      final trip = await widget.api.findActiveTrip();
      if (!mounted) return;
      setState(() => _activeTrip = trip);
    } catch (_) {
      // Best-effort - tap just falls back to normal behavior if this fails.
    }
  }

  Future<void> _addVehicle() async {
    final created = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => VehicleProfileScreen(api: widget.api)),
    );
    if (created == true) _reload();
  }

  Future<void> _editVehicle(VehicleProfile v) async {
    final edited = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => VehicleProfileScreen(api: widget.api, existing: v)),
    );
    if (edited == true) _reload();
  }

  Future<void> _deleteVehicle(VehicleProfile v) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Obriši vozilo?'),
        content: const Text('Ova radnja se ne može opozvati.'),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text('Otkaži')),
          TextButton(onPressed: () => Navigator.of(context).pop(true), child: const Text('Obriši')),
        ],
      ),
    );
    if (confirmed != true) return;

    try {
      await widget.api.deleteVehicle(v.id!);
      _reload();
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Greška: ${e.message}')));
    }
  }

  bool get _isDispatcher => widget.api.role == 'dispatcher';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(_isDispatcher ? 'Vozila u floti' : 'Moja vozila')),
      body: Stack(
        children: [
          Column(
            children: [
              EmailVerificationBanner(api: widget.api),
              ActiveTripBanner(api: widget.api),
              Expanded(
                child: FutureBuilder<List<VehicleProfile>>(
                  future: _vehiclesFuture,
                  builder: (context, snapshot) {
                    if (snapshot.connectionState == ConnectionState.waiting) {
                      return const Center(child: CircularProgressIndicator());
                    }
                    if (snapshot.hasError) {
                      return Center(child: Text('Greška: ${snapshot.error}'));
                    }
                    final vehicles = snapshot.data ?? [];
                    if (vehicles.isEmpty) {
                      return Center(
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(_isDispatcher ? 'Nemate još nijedno vozilo u floti.' : 'Nemaš još sačuvano vozilo.'),
                            const SizedBox(height: 12),
                            FilledButton(onPressed: _addVehicle, child: const Text('Dodaj vozilo')),
                          ],
                        ),
                      );
                    }
                    return ListView.builder(
                      itemCount: vehicles.length,
                      itemBuilder: (context, i) {
                        final v = vehicles[i];
                        // A managed driver's list mixes their own personal
                        // vehicles with their dispatcher's fleet (see
                        // documentations/features/ dispatcher/driver roles entry)
                        // - two vehicles can easily look identical (same demo
                        // default profile) without this label, and only the
                        // OWNER (self for personal, the dispatcher for fleet) may
                        // edit/delete (backend: vehicleMutable, stricter than the
                        // read-only vehicleAccessible).
                        final canManage = !v.isFleet || widget.api.role == 'dispatcher';
                        return ListTile(
                          leading: Icon(v.isFleet ? Icons.warehouse_outlined : Icons.local_shipping),
                          title: Text('${v.heightM}m / ${v.widthM}m / ${v.lengthM}m'),
                          subtitle: Text(
                            '${v.weightKg.toStringAsFixed(0)}kg${v.hazmat ? " · hazmat" : ""}${v.isFleet ? " · Flota" : ""}',
                          ),
                          trailing: canManage
                              ? PopupMenuButton<String>(
                                  onSelected: (action) {
                                    if (action == 'edit') _editVehicle(v);
                                    if (action == 'delete') _deleteVehicle(v);
                                  },
                                  itemBuilder: (context) => const [
                                    PopupMenuItem(value: 'edit', child: Text('Izmeni')),
                                    PopupMenuItem(value: 'delete', child: Text('Obriši')),
                                  ],
                                )
                              : null,
                          // A managed driver's dispatcher creates trips for
                          // FLEET vehicles (backend rejects self-service
                          // POST /trips against a fleet truck for a managed
                          // driver) - tapping one here just opens edit, same
                          // as the popup menu's "Izmeni". A dispatcher
                          // doesn't drive either - tapping a fleet vehicle
                          // just opens edit too. But a managed driver's OWN
                          // personal vehicle still goes to RouteRequestScreen,
                          // same as an independent driver (the backend allows
                          // self-service there, gated on no pending dispatcher
                          // offer / no own trip already under way).
                          // Takes priority over all of that: if THIS vehicle is
                          // the one on the active trip, go straight to
                          // ActiveTripScreen instead - RouteRequestScreen would
                          // just show the "already have an active trip" block.
                          onTap: (_activeTrip != null && _activeTrip!.vehicleId == v.id)
                              ? () async {
                                  await Navigator.of(context).push(
                                    MaterialPageRoute(
                                        builder: (_) =>
                                            ActiveTripScreen(api: widget.api, trip: _activeTrip!, vehicleId: v.id!)),
                                  );
                                  _loadActiveTrip();
                                }
                              : _isDispatcher
                                  ? () => _editVehicle(v)
                                  : (widget.api.dispatcherId != null && v.isFleet)
                                      ? () => ScaffoldMessenger.of(context).showSnackBar(
                                            const SnackBar(
                                                content: Text(
                                                    'Vaš dispečer kreira ture za flotna vozila - ovde samo upravljate vozilima.')),
                                          )
                                      : () => Navigator.of(context).push(
                                            MaterialPageRoute(
                                                builder: (_) => RouteRequestScreen(api: widget.api, vehicle: v)),
                                          ),
                        );
                      },
                    );
                  },
                ),
              ),
            ],
          ),
          // A managed driver or a dispatcher reaches this screen as a
          // SECONDARY destination (an item inside their own home screen's
          // RadialFabMenu, which already offers Profil/Preference/...) -
          // showing a second, identical menu here would just be a confusing
          // duplicate. Only a self-service driver (no dispatcher, not a
          // dispatcher account), for whom this screen IS the main hub, gets one.
          if (!_isDispatcher && widget.api.dispatcherId == null)
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
      floatingActionButton:
          FloatingActionButton(onPressed: _addVehicle, tooltip: 'Dodaj vozilo', child: const Icon(Icons.add)),
    );
  }
}
