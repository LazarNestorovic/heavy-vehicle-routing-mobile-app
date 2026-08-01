import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../models/trip.dart';
import '../services/api_client.dart';
import '../services/trip_socket.dart';
import '../theme/nocturne_theme.dart';

/// Dispatcher's live fleet map - GET /api/v1/trips?status=created and
/// ?status=in_progress for the list of currently active assigned trips (both
/// statuses count as "on the road" - see the comment on _load() for why),
/// then one TripSocket connection per trip (reusing the existing per-trip WS
/// gateway as-is - see documentations/features/ entry for why this is
/// client-side aggregation rather than a new server-side broadcast gateway).
/// Only shows drivers currently on an active trip; there's no idle-position
/// tracking in this project (position is only reported per-trip while one is
/// active, not a standing GPS feed).
class DispatcherLiveMapScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherLiveMapScreen({super.key, required this.api});

  @override
  State<DispatcherLiveMapScreen> createState() => _DispatcherLiveMapScreenState();
}

class _DispatcherLiveMapScreenState extends State<DispatcherLiveMapScreen> {
  final Map<int, TripSocket> _sockets = {};
  final Map<int, LatLng> _positions = {};
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      // A trip stays "created" until the async trip.started worker flips it
      // to "in_progress" (rest-stop suggestion computed) - that can lag well
      // behind the driver actually being on the road, so both statuses count
      // as "live" here (same as HasActiveTrip/TripListScreen's Pokrenute tab)
      // or a just-started trip would be invisible on this map until the
      // worker catches up.
      final results = await Future.wait([
        widget.api.listMyTrips(status: 'created'),
        widget.api.listMyTrips(status: 'in_progress'),
      ]);
      final trips = [...results[0], ...results[1]];
      if (!mounted) return;
      setState(() => _loading = false);
      for (final trip in trips) {
        _watch(trip);
      }
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = 'Greška: $e';
        _loading = false;
      });
    }
  }

  void _watch(Trip trip) {
    final socket = TripSocket();
    _sockets[trip.id] = socket;
    socket.connect(trip.id, widget.api.token ?? '').listen(
      (update) {
        if (!mounted) return;
        setState(() {
          if (update.status == 'arrived') {
            _positions.remove(trip.id);
          } else {
            _positions[trip.id] = LatLng(update.lat, update.lon);
          }
        });
      },
      onError: (_) {}, // one trip's socket dropping shouldn't take down the whole map
    );
  }

  @override
  void dispose() {
    for (final socket in _sockets.values) {
      socket.close();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Vozila uživo')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : Column(
              children: [
                if (_error != null)
                  Padding(
                    padding: const EdgeInsets.all(8),
                    child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                  ),
                if (_positions.isEmpty)
                  const Padding(
                    padding: EdgeInsets.all(16),
                    child: Text('Nema vozila trenutno na putu.'),
                  ),
                Expanded(
                  child: GoogleMap(
                    initialCameraPosition: const CameraPosition(target: LatLng(44.5, 20.5), zoom: 7),
                    markers: _positions.entries
                        .map((entry) => Marker(
                              markerId: MarkerId('trip_${entry.key}'),
                              position: entry.value,
                              icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueViolet),
                              infoWindow: InfoWindow(title: 'Tura #${entry.key}'),
                            ))
                        .toSet(),
                  ),
                ),
              ],
            ),
    );
  }
}
