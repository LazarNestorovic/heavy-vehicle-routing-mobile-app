import 'rest_stop.dart';

/// One message pushed over GET /ws/trips/{id}.
class PositionUpdate {
  final double lat;
  final double lon;
  final double progressFraction;
  final double etaMin;
  final String status; // "in_progress" | "arrived"
  final RestStop? restStop;

  const PositionUpdate({
    required this.lat,
    required this.lon,
    required this.progressFraction,
    required this.etaMin,
    required this.status,
    this.restStop,
  });

  factory PositionUpdate.fromJson(Map<String, dynamic> json) => PositionUpdate(
        lat: (json['lat'] as num).toDouble(),
        lon: (json['lon'] as num).toDouble(),
        progressFraction: (json['progress_fraction'] as num).toDouble(),
        etaMin: (json['eta_min'] as num).toDouble(),
        status: json['status'] as String? ?? 'in_progress',
        restStop: json['rest_stop'] != null ? RestStop.fromJson(json['rest_stop'] as Map<String, dynamic>) : null,
      );
}
