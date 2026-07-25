import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../models/trip.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/polyline.dart';
import '../theme/nocturne_theme.dart';
import 'active_trip_screen.dart';

/// Shows a dispatcher-assigned trip's route, cargo, and vehicle before the
/// driver commits to it - addresses a real gap found in live testing: the
/// driver previously had no way to review a trip before starting it. Handles
/// all three points in the offered->accepted->started state machine (see
/// documentations/features/ entry for the dispatcher/driver roles feature):
///   - "offered": [Odbij] [Prihvati]
///   - "accepted": [Kreni]
class TripDetailScreen extends StatefulWidget {
  final ApiClient api;
  final Trip trip;
  const TripDetailScreen({super.key, required this.api, required this.trip});

  @override
  State<TripDetailScreen> createState() => _TripDetailScreenState();
}

class _TripDetailScreenState extends State<TripDetailScreen> {
  late final List<LatLng> _routePoints;
  late Future<VehicleProfile> _vehicleFuture;
  Trip? _updatedTrip;
  bool _acting = false;
  String? _error;

  Trip get _trip => _updatedTrip ?? widget.trip;

  @override
  void initState() {
    super.initState();
    _routePoints = decodePolyline6(widget.trip.shape);
    _vehicleFuture = widget.api.getVehicle(widget.trip.vehicleId);
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
        MaterialPageRoute(builder: (_) => ActiveTripScreen(api: widget.api, trip: started, vehicleId: started.vehicleId)),
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
            child: FlutterMap(
              options: MapOptions(
                initialCameraFit: _routePoints.isEmpty
                    ? null
                    : CameraFit.bounds(bounds: LatLngBounds.fromPoints(_routePoints), padding: const EdgeInsets.all(32)),
                initialCenter: _routePoints.isEmpty ? const LatLng(44.5, 20.5) : _routePoints.first,
                initialZoom: 7,
                interactionOptions: const InteractionOptions(flags: InteractiveFlag.all & ~InteractiveFlag.rotate),
              ),
              children: [
                TileLayer(
                  urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                  userAgentPackageName: 'com.example.hvr_mobile',
                ),
                if (_routePoints.isNotEmpty)
                  PolylineLayer(polylines: [
                    Polyline(points: _routePoints, strokeWidth: 4, color: NocturneColors.accent),
                  ]),
                MarkerLayer(markers: [
                  if (_routePoints.isNotEmpty)
                    Marker(
                      point: _routePoints.first,
                      width: 32,
                      height: 32,
                      child: const Icon(Icons.trip_origin, color: Colors.green),
                    ),
                  if (_routePoints.isNotEmpty)
                    Marker(
                      point: _routePoints.last,
                      width: 32,
                      height: 32,
                      child: const Icon(Icons.flag, color: Colors.red),
                    ),
                ]),
              ],
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
                          Text('${_trip.distanceKm.toStringAsFixed(1)} km · ${_trip.durationMin.toStringAsFixed(0)} min',
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
      return FilledButton(
        onPressed: _acting ? null : _start,
        child: _acting
            ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
            : const Text('Kreni'),
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
