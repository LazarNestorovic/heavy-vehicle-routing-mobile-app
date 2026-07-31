import 'dart:async';

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../models/trip.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/location_service.dart';
import '../services/polyline.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/start_proximity_status.dart';
import 'active_trip_screen.dart';

/// Shows a dispatcher-assigned trip's route, cargo, and vehicle before the
/// driver commits to it - addresses a real gap found in live testing: the
/// driver previously had no way to review a trip before starting it. Handles
/// all three points in the offered->accepted->started state machine (see
/// documentations/features/ entry for the dispatcher/driver roles feature):
///   - "offered": [Odbij] [Prihvati]
///   - "accepted": [Kreni] - gated on being at the route's origin (see
///     widgets/start_proximity_status.dart), same as RouteRequestScreen.
class TripDetailScreen extends StatefulWidget {
  final ApiClient api;
  final Trip trip;
  const TripDetailScreen({super.key, required this.api, required this.trip});

  @override
  State<TripDetailScreen> createState() => _TripDetailScreenState();
}

class _TripDetailScreenState extends State<TripDetailScreen> {
  final _locationService = LocationService();
  StreamSubscription<Position>? _positionSub;
  LatLng? _myPosition;

  late final List<LatLng> _routePoints;
  late Future<VehicleProfile> _vehicleFuture;
  Trip? _updatedTrip;
  bool _acting = false;
  bool _canStart = false;
  String? _error;

  // Blocks starting THIS trip while another one is already active - mirrors
  // the backend's own 409 in handleStartTrip (see widgets/active_trip_banner.dart
  // for the other half: a way back to an active trip from the home screen).
  Trip? _activeTrip;

  Trip get _trip => _updatedTrip ?? widget.trip;

  @override
  void initState() {
    super.initState();
    _routePoints = decodePolyline6(widget.trip.shape);
    _vehicleFuture = widget.api.getVehicle(widget.trip.vehicleId);
    // Shows the driver's own position on the map even before starting the
    // trip (not just once live-tracking kicks in on ActiveTripScreen) - the
    // driver asked to always see themselves on the map, not only mid-route.
    unawaited(_watchPosition());
    unawaited(_checkActiveTrip());
  }

  Future<void> _checkActiveTrip() async {
    try {
      final trip = await widget.api.findActiveTrip();
      if (!mounted) return;
      setState(() => _activeTrip = trip);
    } catch (_) {
      // Best-effort - if the check itself fails, don't block on it; the
      // backend's own 409 is still the real enforcement either way.
    }
  }

  Future<void> _watchPosition() async {
    if (await _locationService.ensurePermission() != GpsStatus.granted) return;
    _positionSub = _locationService.positionStream().listen((p) {
      if (!mounted) return;
      setState(() => _myPosition = LatLng(p.latitude, p.longitude));
    });
  }

  @override
  void dispose() {
    _positionSub?.cancel();
    super.dispose();
  }

  // GoogleMap has no declarative "fit these points" option (unlike flutter_map's
  // initialCameraFit) - compute bounds ourselves and fit them once the map/
  // controller exists.
  LatLngBounds? get _routeBounds {
    if (_routePoints.isEmpty) return null;
    var minLat = _routePoints.first.latitude, maxLat = _routePoints.first.latitude;
    var minLon = _routePoints.first.longitude, maxLon = _routePoints.first.longitude;
    for (final p in _routePoints) {
      minLat = p.latitude < minLat ? p.latitude : minLat;
      maxLat = p.latitude > maxLat ? p.latitude : maxLat;
      minLon = p.longitude < minLon ? p.longitude : minLon;
      maxLon = p.longitude > maxLon ? p.longitude : maxLon;
    }
    return LatLngBounds(southwest: LatLng(minLat, minLon), northeast: LatLng(maxLat, maxLon));
  }

  void _onMapCreated(GoogleMapController controller) {
    final bounds = _routeBounds;
    if (bounds != null) {
      controller.animateCamera(CameraUpdate.newLatLngBounds(bounds, 32));
    }
  }

  Future<void> _accept() async {
    setState(() {
      _acting = true;
      _error = null;
    });
    try {
      final updated = await widget.api.acceptTrip(_trip.id);
      if (!mounted) return;
      setState(() => _updatedTrip = updated);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Greška: $e');
    } finally {
      if (mounted) setState(() => _acting = false);
    }
  }

  Future<void> _reject() async {
    setState(() {
      _acting = true;
      _error = null;
    });
    try {
      await widget.api.rejectTrip(_trip.id);
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = 'Greška: $e';
        _acting = false;
      });
    }
  }

  Future<void> _start() async {
    setState(() {
      _acting = true;
      _error = null;
    });
    try {
      final started = await widget.api.startTrip(_trip.id);
      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(
            builder: (_) => ActiveTripScreen(api: widget.api, trip: started, vehicleId: started.vehicleId)),
      );
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = 'Greška: $e';
        _acting = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('Detalji ture #${_trip.id}')),
      body: Column(
        children: [
          Expanded(
            flex: 3,
            child: GoogleMap(
              onMapCreated: _onMapCreated,
              initialCameraPosition:
                  CameraPosition(target: _routePoints.isEmpty ? const LatLng(44.5, 20.5) : _routePoints.first, zoom: 7),
              rotateGesturesEnabled: false,
              polylines: {
                if (_routePoints.isNotEmpty)
                  Polyline(
                      polylineId: const PolylineId('route'),
                      points: _routePoints,
                      width: 4,
                      color: NocturneColors.accent),
              },
              markers: {
                if (_myPosition != null)
                  Marker(
                    markerId: const MarkerId('my_position'),
                    position: _myPosition!,
                    icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueAzure),
                    anchor: const Offset(0.5, 0.5),
                    flat: true,
                    zIndex: 2,
                    infoWindow: const InfoWindow(title: 'Vaša pozicija'),
                  ),
                if (_routePoints.isNotEmpty)
                  Marker(
                    markerId: const MarkerId('origin'),
                    position: _routePoints.first,
                    icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueGreen),
                  ),
                if (_routePoints.isNotEmpty)
                  Marker(
                    markerId: const MarkerId('destination'),
                    position: _routePoints.last,
                    icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueRed),
                  ),
              },
            ),
          ),
          Expanded(
            flex: 2,
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Card(
                    margin: const EdgeInsets.all(12),
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                              '${_trip.distanceKm.toStringAsFixed(1)} km · ${_trip.durationMin.toStringAsFixed(0)} min',
                              style: Theme.of(context).textTheme.titleMedium),
                          if (_trip.explanation != null) ...[
                            const SizedBox(height: 6),
                            Text(_trip.explanation!, style: const TextStyle(fontStyle: FontStyle.italic)),
                          ],
                        ],
                      ),
                    ),
                  ),
                  if (_trip.cargoDescription != null ||
                      _trip.cargoWeightKg != null ||
                      _trip.cargoTempRange != null ||
                      _trip.pickupLocation != null ||
                      _trip.dropoffLocation != null)
                    Card(
                      margin: const EdgeInsets.symmetric(horizontal: 12),
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Tovar', style: Theme.of(context).textTheme.titleSmall),
                            const SizedBox(height: 4),
                            if (_trip.cargoDescription != null) Text(_trip.cargoDescription!),
                            if (_trip.cargoWeightKg != null) Text('${_trip.cargoWeightKg!.toStringAsFixed(0)} kg'),
                            if (_trip.cargoTempRange != null) Text(_trip.cargoTempRange!),
                            if (_trip.pickupLocation != null) Text('Preuzimanje: ${_trip.pickupLocation}'),
                            if (_trip.dropoffLocation != null) Text('Isporuka: ${_trip.dropoffLocation}'),
                          ],
                        ),
                      ),
                    ),
                  FutureBuilder<VehicleProfile>(
                    future: _vehicleFuture,
                    builder: (context, snapshot) {
                      if (!snapshot.hasData) {
                        return const Padding(padding: EdgeInsets.all(16), child: LinearProgressIndicator());
                      }
                      final v = snapshot.data!;
                      return Card(
                        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                        child: ListTile(
                          leading: const Icon(Icons.local_shipping),
                          title: Text('${v.heightM}m / ${v.widthM}m / ${v.lengthM}m'),
                          subtitle: Text('${v.weightKg.toStringAsFixed(0)}kg${v.hazmat ? " · hazmat" : ""}'),
                        ),
                      );
                    },
                  ),
                  if (_error != null)
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                    ),
                  Padding(
                    padding: const EdgeInsets.all(12),
                    child: _buildActions(context),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildActions(BuildContext context) {
    if (_trip.status == 'accepted') {
      final activeTrip = _activeTrip;
      if (activeTrip != null) {
        return Card(
          color: NocturneColors.accent800,
          child: Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text('Već imate aktivnu turu u toku - prvo je završite.'),
                const SizedBox(height: 8),
                FilledButton(
                  onPressed: () => Navigator.of(context).push(
                    MaterialPageRoute(
                        builder: (_) =>
                            ActiveTripScreen(api: widget.api, trip: activeTrip, vehicleId: activeTrip.vehicleId)),
                  ),
                  child: const Text('Idi na nju'),
                ),
              ],
            ),
          ),
        );
      }
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_routePoints.isNotEmpty)
            StartProximityStatus(
              originLat: _routePoints.first.latitude,
              originLon: _routePoints.first.longitude,
              onCanStartChanged: (canStart) => setState(() => _canStart = canStart),
            ),
          FilledButton(
            onPressed: (_acting || !_canStart) ? null : _start,
            child: _acting
                ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                : const Text('Kreni'),
          ),
        ],
      );
    }
    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: _acting ? null : _reject,
            child: const Text('Odbij'),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: FilledButton(
            onPressed: _acting ? null : _accept,
            child: _acting
                ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                : const Text('Prihvati'),
          ),
        ),
      ],
    );
  }
}
