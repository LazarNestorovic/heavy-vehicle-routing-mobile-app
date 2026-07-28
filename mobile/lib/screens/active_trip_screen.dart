import 'dart:async';

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../models/position_update.dart';
import '../models/rest_stop.dart';
import '../models/trip.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/location_service.dart';
import '../services/polyline.dart';
import '../services/trip_socket.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/radial_fab_menu.dart';
import 'cargo_screen.dart';
import 'chat_list_screen.dart';
import 'profile_screen.dart';
import 'trip_log_screen.dart';
import 'truck_status_screen.dart';

/// Active trip screen (SPECIFIKACIJA.md 3.7/3.9/3.10): live GPS position over
/// WebSocket, ETA, and a rest-stop alert the first time the worker attaches
/// one. See documentations/fixes/2026-07-28-remove-simulated-fallback.md -
/// there's no simulated fallback if GPS isn't available, only the
/// [_gpsStatusBanner] explaining why and offering a fix.
/// The WS endpoint now requires auth too (?token= query param, since browsers'
/// WebSocket API can't set custom headers - see backend/internal/httpapi/middleware.go
/// RequireAuthQuery), hence `api` is needed here just for its token.
///
/// This is also the mock-up's "Main" screen (see documentations/features/
/// 2026-07-21-nocturne-redesign.md) - only `vehicleId` is required (not a full
/// VehicleProfile): a dispatcher-assigned trip reaches this screen via
/// OfferedTripsScreen, which never has the vehicle object in hand the way
/// RouteRequestScreen does, so this screen loads it itself. Profile/Truck
/// Status/Cargo/Trip Log/Chat are reached via the draggable RadialFabMenu
/// overlaid on the map, matching the mock-up.
class ActiveTripScreen extends StatefulWidget {
  final ApiClient api;
  final Trip trip;
  final int vehicleId;
  const ActiveTripScreen({super.key, required this.api, required this.trip, required this.vehicleId});

  @override
  State<ActiveTripScreen> createState() => _ActiveTripScreenState();
}

class _ActiveTripScreenState extends State<ActiveTripScreen> {
  final _socket = TripSocket();
  final _locationService = LocationService();
  StreamSubscription<Position>? _positionSub;
  bool _liveGpsActive = false;
  GpsStatus? _gpsStatus;
  GoogleMapController? _mapController;

  late final List<LatLng> _routePoints;
  LatLng? _currentPosition;
  double _progressFraction = 0;
  late double _etaMin;
  String _status = 'in_progress';
  RestStop? _restStop;
  bool _restStopAlertShown = false;
  String? _error;
  int _chatUnreadTotal = 0;
  VehicleProfile? _vehicle;

  @override
  void initState() {
    super.initState();
    _routePoints = decodePolyline6(widget.trip.shape);
    _etaMin = widget.trip.durationMin;
    if (_routePoints.isNotEmpty) {
      _currentPosition = _routePoints.first;
    }
    if (widget.trip.explanation != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _showExplanationBanner(widget.trip.explanation!);
      });
    }
    _listen();
    _loadChatUnreadTotal();
    _loadVehicle();
    _startLiveGps();
  }

  // Starts reporting real GPS fixes to the backend if the user grants
  // location permission - see services/location_service.dart and
  // backend/internal/ws/gateway.go. If permission ISN'T granted (denied,
  // denied forever, or the phone's location toggle is off), there is no
  // fallback anymore (see documentations/fixes/2026-07-28-remove-simulated-
  // fallback.md) - _gpsStatus instead drives a banner (see build()) telling
  // the driver exactly why their position isn't showing and offering a
  // one-tap fix. In practice GPS permission is already required to start a
  // trip at all (see widgets/start_proximity_status.dart), so this mainly
  // covers permission being revoked mid-trip.
  Future<void> _startLiveGps() async {
    final status = await _locationService.ensurePermission();
    if (!mounted) return;
    setState(() => _gpsStatus = status);
    if (status != GpsStatus.granted) return;

    setState(() => _liveGpsActive = true);
    _positionSub = _locationService.positionStream().listen((position) {
      if (!mounted) return;
      final point = LatLng(position.latitude, position.longitude);
      setState(() => _currentPosition = point);
      _mapController?.animateCamera(CameraUpdate.newLatLng(point));

      // Fire-and-forget: a dropped ping just means this update didn't reach
      // the dispatcher's live map, the next one (20m later) will.
      unawaited(widget.api.reportPosition(widget.trip.id, position.latitude, position.longitude).catchError((_) {}));
    });
  }

  Future<void> _completeTrip() async {
    try {
      await widget.api.completeTrip(widget.trip.id);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Greška: $e')));
    }
  }

  Future<void> _loadVehicle() async {
    try {
      final vehicle = await widget.api.getVehicle(widget.vehicleId);
      if (!mounted) return;
      setState(() => _vehicle = vehicle);
    } catch (_) {
      // Non-fatal: Truck Status just won't be openable until this succeeds.
    }
  }

  // Refreshed once on entry (and again whenever the chat list screen is
  // popped back to here) - not live-updating on its own, same scope cut as
  // the rest of chat's polling-free design (REST is the source of truth).
  Future<void> _loadChatUnreadTotal() async {
    try {
      final chats = await widget.api.listChats();
      if (!mounted) return;
      setState(() => _chatUnreadTotal = chats.fold(0, (sum, c) => sum + c.unreadCount));
    } catch (_) {
      // Non-critical - the badge just stays at its last known value.
    }
  }

  void _listen() {
    _socket.connect(widget.trip.id, widget.api.token ?? '').listen(
      (update) => _onUpdate(update),
      onError: (e) => setState(() => _error = 'Konekcija prekinuta: $e'),
      onDone: () {
        if (mounted && _status != 'arrived') {
          setState(() => _error = 'Konekcija zatvorena.');
        }
      },
    );
  }

  void _onUpdate(PositionUpdate update) {
    if (!mounted) return;
    setState(() {
      _currentPosition = LatLng(update.lat, update.lon);
      _progressFraction = update.progressFraction;
      _etaMin = update.etaMin;
      _status = update.status;
      if (update.restStop != null && !_restStopAlertShown) {
        _restStop = update.restStop;
        _restStopAlertShown = true;
        _showRestStopAlert(update.restStop!);
      }
    });
    _mapController?.animateCamera(CameraUpdate.newLatLng(_currentPosition!));

    if (update.status == 'arrived') {
      _showArrivedDialog();
    }
  }

  // Explains why the driver's position isn't being shown/reported, with a
  // one-tap fix where one exists - see _startLiveGps's doc comment.
  Widget _gpsStatusBanner() {
    late final String message;
    late final String actionLabel;
    late final Future<void> Function() action;
    switch (_gpsStatus!) {
      case GpsStatus.serviceDisabled:
        message = 'GPS je isključen na telefonu - vaša pozicija se ne prikazuje.';
        actionLabel = 'Uključi lokaciju';
        action = _locationService.openLocationSettings;
        break;
      case GpsStatus.deniedForever:
        message = 'Dozvola za lokaciju je trajno odbijena - vaša pozicija se ne prikazuje.';
        actionLabel = 'Podešavanja';
        action = _locationService.openAppSettings;
        break;
      case GpsStatus.denied:
        message = 'Dozvola za lokaciju nije data - vaša pozicija se ne prikazuje.';
        actionLabel = 'Pokušaj ponovo';
        action = _startLiveGps;
        break;
      case GpsStatus.granted:
        return const SizedBox.shrink(); // unreachable (guarded by the caller), keeps the switch exhaustive
    }

    return Container(
      width: double.infinity,
      color: NocturneColors.accent800,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        children: [
          const Icon(Icons.gps_off, size: 18, color: NocturneColors.accent300),
          const SizedBox(width: 8),
          Expanded(child: Text(message)),
          TextButton(onPressed: () => action(), child: Text(actionLabel)),
        ],
      ),
    );
  }

  void _showExplanationBanner(String explanation) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(explanation), duration: const Duration(seconds: 6)),
    );
  }

  void _showRestStopAlert(RestStop stop) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('Predlog pauze: ${stop.label} (${stop.amenityLabel})'),
        duration: const Duration(seconds: 8),
        backgroundColor: NocturneColors.accent800,
      ),
    );
  }

  void _showArrivedDialog() {
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Stigli ste'),
        content: const Text('Vozilo je stiglo na odredište.'),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.of(context).pop();
              Navigator.of(context).pop();
            },
            child: const Text('U redu'),
          ),
        ],
      ),
    );
  }

  @override
  void dispose() {
    _positionSub?.cancel();
    _socket.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Putovanje #${widget.trip.id}'),
        actions: [
          if (_liveGpsActive && _status != 'arrived')
            IconButton(
              icon: const Icon(Icons.flag_outlined),
              tooltip: 'Stigao sam',
              onPressed: _completeTrip,
            ),
        ],
      ),
      body: Column(
        children: [
          if (_error != null)
            Container(
              width: double.infinity,
              color: NocturneColors.error.withValues(alpha: 0.15),
              padding: const EdgeInsets.all(8),
              child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
            ),
          if (_gpsStatus != null && _gpsStatus != GpsStatus.granted) _gpsStatusBanner(),
          _StatusBar(
            progressFraction: _progressFraction,
            etaMin: _etaMin,
            status: _status,
            restStop: _restStop,
          ),
          Expanded(
            child: Stack(
              children: [
                GoogleMap(
                  onMapCreated: (controller) => _mapController = controller,
                  initialCameraPosition: CameraPosition(target: _currentPosition ?? const LatLng(44.5, 20.5), zoom: 9),
                  polylines: {
                    Polyline(
                      polylineId: const PolylineId('route'),
                      points: _routePoints,
                      width: 4,
                      color: NocturneColors.accent.withValues(alpha: 0.5),
                    ),
                  },
                  markers: {
                    if (_currentPosition != null)
                      Marker(
                        markerId: const MarkerId('truck'),
                        position: _currentPosition!,
                        icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueViolet),
                      ),
                    if (_restStop != null)
                      Marker(
                        markerId: const MarkerId('rest_stop'),
                        position: LatLng(_restStop!.lat, _restStop!.lon),
                        icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueOrange),
                      ),
                  },
                ),
                RadialFabMenu(
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
                      tooltip: 'Status vozila',
                      onTap: () {
                        if (_vehicle == null) {
                          ScaffoldMessenger.of(context)
                              .showSnackBar(const SnackBar(content: Text('Podaci o vozilu se još učitavaju...')));
                          return;
                        }
                        Navigator.of(context).push(
                          MaterialPageRoute(
                            builder: (_) => TruckStatusScreen(api: widget.api, vehicle: _vehicle!, currentTrip: widget.trip),
                          ),
                        );
                      },
                    ),
                    RadialFabMenuItem(
                      icon: Icons.inventory_2_outlined,
                      tooltip: 'Tovar',
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => CargoScreen(trip: widget.trip)),
                      ),
                    ),
                    RadialFabMenuItem(
                      icon: Icons.history,
                      tooltip: 'Dnevnik putovanja',
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => TripLogScreen(api: widget.api, tripId: widget.trip.id)),
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
          ),
        ],
      ),
    );
  }
}

class _StatusBar extends StatelessWidget {
  final double progressFraction;
  final double etaMin;
  final String status;
  final RestStop? restStop;

  const _StatusBar({
    required this.progressFraction,
    required this.etaMin,
    required this.status,
    required this.restStop,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          LinearProgressIndicator(value: progressFraction.clamp(0.0, 1.0).toDouble()),
          const SizedBox(height: 6),
          Text(
            status == 'arrived' ? 'Stigli ste' : 'ETA: ${etaMin.toStringAsFixed(0)} min',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          if (restStop != null)
            Text('Sledeća pauza: ${restStop!.label}', style: const TextStyle(color: NocturneColors.accent300)),
        ],
      ),
    );
  }
}
