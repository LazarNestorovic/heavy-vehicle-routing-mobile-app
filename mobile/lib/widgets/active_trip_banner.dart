import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../screens/active_trip_screen.dart';
import '../services/api_client.dart';
import '../services/route_observer.dart';
import '../theme/nocturne_theme.dart';

/// Shown on a driver's home screen when they have a trip already "created" or
/// "in_progress" - lets them jump back into ActiveTripScreen after leaving it
/// (e.g. via the back button), instead of it only being reachable once, right
/// when the trip starts. Renders nothing if there's no active trip. The
/// backend independently rejects starting a SECOND trip while one is active
/// (see backend handleCreateTrip/handleStartTrip) - this banner is the other
/// half of that: a way back to the one they already have, not a way around
/// the block.
class ActiveTripBanner extends StatefulWidget {
  final ApiClient api;
  const ActiveTripBanner({super.key, required this.api});

  @override
  State<ActiveTripBanner> createState() => _ActiveTripBannerState();
}

class _ActiveTripBannerState extends State<ActiveTripBanner> with RouteAware {
  Trip? _trip;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
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

  // Called when a route pushed on top of the screen hosting this banner (e.g.
  // ActiveTripScreen, or RouteRequestScreen while starting a new trip) is
  // popped and that screen becomes visible again - Flutter does NOT call
  // initState() again for it, so without this the banner kept showing
  // whatever was true when it was first built (e.g. "no active trip") even
  // after a trip was started/completed in the meantime.
  @override
  void didPopNext() => _load();

  Future<void> _load() async {
    // A dispatcher account has no trips of their own to resume - GET /trips
    // for them lists trips they ASSIGNED (by assigned_by_id), so calling
    // findActiveTrip() here would wrongly surface one of their drivers'
    // active trips as if it were the dispatcher's own to drive.
    if (widget.api.role == 'dispatcher') {
      if (mounted) setState(() => _loading = false);
      return;
    }
    try {
      final trip = await widget.api.findActiveTrip();
      if (!mounted) return;
      setState(() {
        _trip = trip;
        _loading = false;
      });
    } catch (_) {
      // Best-effort - banner just stays hidden until the next reload.
      if (!mounted) return;
      setState(() => _loading = false);
    }
  }

  Future<void> _resume() async {
    final trip = _trip;
    if (trip == null) return;
    await Navigator.of(context).push(
      MaterialPageRoute(builder: (_) => ActiveTripScreen(api: widget.api, trip: trip, vehicleId: trip.vehicleId)),
    );
    // The trip may have been completed while we were away - refresh so the
    // banner disappears instead of offering a stale "resume". (didPopNext
    // above also fires for this same pop and would cover it too - calling
    // explicitly here as well is harmless, just a redundant refresh.)
    if (mounted) _load();
  }

  @override
  Widget build(BuildContext context) {
    if (_loading || _trip == null) return const SizedBox.shrink();

    return Card(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      color: NocturneColors.accent800,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            const Icon(Icons.local_shipping, color: NocturneColors.accent300),
            const SizedBox(width: 8),
            const Expanded(child: Text('Imate aktivnu turu u toku.')),
            FilledButton(onPressed: _resume, child: const Text('Nastavi')),
          ],
        ),
      ),
    );
  }
}
