import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../models/position_update.dart';
import '../models/rest_stop.dart';
import '../models/trip.dart';
import '../services/api_client.dart';
import '../services/polyline.dart';
import '../services/trip_socket.dart';

/// Active trip screen (SPECIFIKACIJA.md 3.7/3.9/3.10): live simulated position
/// over WebSocket, ETA, and a rest-stop alert the first time the worker
/// attaches one. See documentations/features/2026-07-21-websocket-gateway.md.
/// The WS endpoint now requires auth too (?token= query param, since browsers'
/// WebSocket API can't set custom headers - see backend/internal/httpapi/middleware.go
/// RequireAuthQuery), hence `api` is needed here just for its token.
class ActiveTripScreen extends StatefulWidget {
  final ApiClient api;
  final Trip trip;
  const ActiveTripScreen({super.key, required this.api, required this.trip});

  @override
  State<ActiveTripScreen> createState() => _ActiveTripScreenState();
}

class _ActiveTripScreenState extends State<ActiveTripScreen> {
  final _socket = TripSocket();
  final _mapController = MapController();

  late final List<LatLng> _routePoints;
  LatLng? _currentPosition;
  double _progressFraction = 0;
  late double _etaMin;
  String _status = 'in_progress';
  RestStop? _restStop;
  bool _restStopAlertShown = false;
  String? _error;

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
    _mapController.move(_currentPosition!, _mapController.camera.zoom);

    if (update.status == 'arrived') {
      _showArrivedDialog();
    }
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
        backgroundColor: Colors.orange.shade800,
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
    _socket.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('Putovanje #${widget.trip.id}')),
      body: Column(
        children: [
          if (_error != null)
            Container(
              width: double.infinity,
              color: Colors.red.shade100,
              padding: const EdgeInsets.all(8),
              child: Text(_error!, style: const TextStyle(color: Colors.red)),
            ),
          _StatusBar(
            progressFraction: _progressFraction,
            etaMin: _etaMin,
            status: _status,
            restStop: _restStop,
          ),
          Expanded(
            child: FlutterMap(
              mapController: _mapController,
              options: MapOptions(
                initialCenter: _currentPosition ?? const LatLng(44.5, 20.5),
                initialZoom: 9,
              ),
              children: [
                TileLayer(
                  urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                  userAgentPackageName: 'com.example.hvr_mobile',
                ),
                PolylineLayer(polylines: [
                  Polyline(points: _routePoints, strokeWidth: 4, color: Colors.indigo.withValues(alpha: 0.5)),
                ]),
                MarkerLayer(markers: [
                  if (_currentPosition != null)
                    Marker(
                      point: _currentPosition!,
                      width: 44,
                      height: 44,
                      child: const Icon(Icons.local_shipping, color: Colors.indigo, size: 32),
                    ),
                  if (_restStop != null)
                    Marker(
                      point: LatLng(_restStop!.lat, _restStop!.lon),
                      width: 36,
                      height: 36,
                      child: const Icon(Icons.local_gas_station, color: Colors.orange),
                    ),
                ]),
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
            Text('Sledeća pauza: ${restStop!.label}', style: const TextStyle(color: Colors.orange)),
        ],
      ),
    );
  }
}
