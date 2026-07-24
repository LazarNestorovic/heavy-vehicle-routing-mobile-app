import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../models/route_result.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/polyline.dart';
import 'active_trip_screen.dart';

/// Map + route request screen (SPECIFIKACIJA.md 3.9): tap once for origin,
/// again for destination, then preview or start the trip. No address search/
/// geocoding in this MVP - tap-to-pick only.
class RouteRequestScreen extends StatefulWidget {
  final ApiClient api;
  final VehicleProfile vehicle;
  const RouteRequestScreen({super.key, required this.api, required this.vehicle});

  @override
  State<RouteRequestScreen> createState() => _RouteRequestScreenState();
}

class _RouteRequestScreenState extends State<RouteRequestScreen> {
  final _mapController = MapController();

  LatLng? _origin;
  LatLng? _destination;
  RouteResult? _routeResult;
  List<LatLng> _routePoints = [];

  bool _loading = false;
  String? _error;

  static const _serbiaCenter = LatLng(44.5, 20.5);

  void _handleTap(TapPosition tapPos, LatLng point) {
    setState(() {
      _routeResult = null;
      _routePoints = [];
      _error = null;
      if (_origin == null) {
        _origin = point;
      } else if (_destination == null) {
        _destination = point;
      } else {
        // Third tap starts over.
        _origin = point;
        _destination = null;
      }
    });
  }

  void _reset() {
    setState(() {
      _origin = null;
      _destination = null;
      _routeResult = null;
      _routePoints = [];
      _error = null;
    });
  }

  Future<void> _previewRoute() async {
    if (_origin == null || _destination == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.api.previewRoute(
        origin: _origin!,
        destination: _destination!,
        vehicle: widget.vehicle,
      );
      setState(() {
        _routeResult = result;
        _routePoints = decodePolyline6(result.shape);
      });
    } on ApiException catch (e) {
      setState(() => _error = 'Greška: ${e.message}');
    } catch (e) {
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _startTrip() async {
    if (_origin == null || _destination == null || widget.vehicle.id == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final trip = await widget.api.createTrip(
        vehicleId: widget.vehicle.id!,
        origin: _origin!,
        destination: _destination!,
      );
      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => ActiveTripScreen(api: widget.api, trip: trip)),
      );
    } on ApiException catch (e) {
      setState(() => _error = 'Greška: ${e.message}');
    } catch (e) {
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Nova ruta'),
        actions: [
          IconButton(onPressed: _reset, icon: const Icon(Icons.refresh), tooltip: 'Resetuj tačke'),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: Text(
              _origin == null
                  ? 'Dodirni mapu da postaviš polaznu tačku.'
                  : _destination == null
                      ? 'Dodirni mapu da postaviš odredište.'
                      : 'Polazna i odredišna tačka su postavljene.',
              textAlign: TextAlign.center,
            ),
          ),
          Expanded(
            child: FlutterMap(
              mapController: _mapController,
              options: MapOptions(
                initialCenter: _serbiaCenter,
                initialZoom: 7,
                onTap: _handleTap,
              ),
              children: [
                TileLayer(
                  urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                  userAgentPackageName: 'com.example.hvr_mobile',
                ),
                if (_routePoints.isNotEmpty)
                  PolylineLayer(polylines: [
                    Polyline(points: _routePoints, strokeWidth: 4, color: Colors.indigo),
                  ]),
                MarkerLayer(markers: [
                  if (_origin != null)
                    Marker(point: _origin!, width: 40, height: 40, child: const Icon(Icons.trip_origin, color: Colors.green)),
                  if (_destination != null)
                    Marker(point: _destination!, width: 40, height: 40, child: const Icon(Icons.flag, color: Colors.red)),
                ]),
              ],
            ),
          ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.all(8),
              child: Text(_error!, style: const TextStyle(color: Colors.red)),
            ),
          if (_routeResult != null) _RouteSummaryCard(result: _routeResult!),
          Padding(
            padding: const EdgeInsets.all(12),
            child: Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: (_origin != null && _destination != null && !_loading) ? _previewRoute : null,
                    child: const Text('Pregled rute'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: (_origin != null && _destination != null && !_loading) ? _startTrip : null,
                    child: _loading
                        ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Kreni na put'),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _RouteSummaryCard extends StatelessWidget {
  final RouteResult result;
  const _RouteSummaryCard({required this.result});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('${result.distanceKm.toStringAsFixed(1)} km · ${result.durationMin.toStringAsFixed(0)} min',
                style: Theme.of(context).textTheme.titleMedium),
            Text('Risk score: ${result.riskScore.toStringAsFixed(1)}'),
            if (result.explanation != null) ...[
              const SizedBox(height: 8),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Icon(Icons.info_outline, size: 18, color: Colors.orange),
                  const SizedBox(width: 6),
                  Expanded(child: Text(result.explanation!, style: const TextStyle(fontStyle: FontStyle.italic))),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}
