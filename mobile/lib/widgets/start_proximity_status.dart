import 'package:flutter/material.dart';

import '../services/location_service.dart';
import '../theme/nocturne_theme.dart';

/// Gates actually STARTING a trip on the driver being at its origin -
/// mirrors turn-by-turn navigation apps (Google Maps etc.), which only let
/// you "start" from where you actually are. Previewing/planning a route
/// stays available regardless (see route_request_screen.dart's "Pregled
/// rute", unaffected by this) - only the "Kreni na put"/"Kreni" action is
/// gated, via [onCanStartChanged]. See documentations/fixes/ entry - this
/// was requested after live GPS tracking (see live-gps-tracking.md) made a
/// mismatch between the planned origin and the driver's real position
/// visibly confusing (the live marker jumping to reflect actual position,
/// far from the drawn route).
class StartProximityStatus extends StatefulWidget {
  static const thresholdMeters = 500.0;

  final double originLat;
  final double originLon;
  final ValueChanged<bool> onCanStartChanged;
  const StartProximityStatus({
    super.key,
    required this.originLat,
    required this.originLon,
    required this.onCanStartChanged,
  });

  @override
  State<StartProximityStatus> createState() => _StartProximityStatusState();
}

class _StartProximityStatusState extends State<StartProximityStatus> {
  final _locationService = LocationService();
  bool _checking = true;
  double? _distanceM;

  @override
  void initState() {
    super.initState();
    _check();
  }

  @override
  void didUpdateWidget(covariant StartProximityStatus oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.originLat != widget.originLat || oldWidget.originLon != widget.originLon) {
      _check();
    }
  }

  Future<void> _check() async {
    setState(() => _checking = true);
    final distance = await _locationService.distanceToCurrentPosition(widget.originLat, widget.originLon);
    if (!mounted) return;
    setState(() {
      _distanceM = distance;
      _checking = false;
    });
    widget.onCanStartChanged(distance != null && distance <= StartProximityStatus.thresholdMeters);
  }

  @override
  Widget build(BuildContext context) {
    if (_checking) {
      return const Padding(
        padding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Row(
          children: [
            SizedBox(height: 14, width: 14, child: CircularProgressIndicator(strokeWidth: 2)),
            SizedBox(width: 8),
            Text('Proveravam udaljenost od polazišta...'),
          ],
        ),
      );
    }

    if (_distanceM == null) {
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        child: Row(
          children: [
            const Icon(Icons.gps_off, size: 18, color: NocturneColors.accent300),
            const SizedBox(width: 8),
            const Expanded(child: Text('Nije moguće proveriti lokaciju - proveri GPS dozvolu.')),
            TextButton(onPressed: _check, child: const Text('Pokušaj ponovo')),
          ],
        ),
      );
    }

    final near = _distanceM! <= StartProximityStatus.thresholdMeters;
    final distanceLabel =
        _distanceM! >= 1000 ? '${(_distanceM! / 1000).toStringAsFixed(1)} km' : '${_distanceM!.toStringAsFixed(0)} m';
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        children: [
          Icon(near ? Icons.check_circle_outline : Icons.location_off_outlined,
              size: 18, color: near ? NocturneColors.accent : NocturneColors.error),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              near ? 'Na polaznoj tački.' : 'Udaljeni ste $distanceLabel od polazišta - morate biti tamo da biste krenuli.',
              style: near ? null : const TextStyle(color: NocturneColors.error),
            ),
          ),
          TextButton(onPressed: _check, child: const Text('Osveži')),
        ],
      ),
    );
  }
}
