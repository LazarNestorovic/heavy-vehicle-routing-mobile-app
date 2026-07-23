import 'route_candidate.dart';

/// Response of POST /api/v1/routes - stateless preview, nothing persisted.
class RouteResult {
  final double distanceKm;
  final double durationMin;
  final String shape;
  final double riskScore;
  final List<RouteCandidate> candidates;
  final String? explanation;

  const RouteResult({
    required this.distanceKm,
    required this.durationMin,
    required this.shape,
    required this.riskScore,
    required this.candidates,
    this.explanation,
  });

  factory RouteResult.fromJson(Map<String, dynamic> json) => RouteResult(
        distanceKm: (json['distance_km'] as num).toDouble(),
        durationMin: (json['duration_min'] as num).toDouble(),
        shape: json['shape'] as String,
        riskScore: (json['risk_score'] as num).toDouble(),
        candidates: (json['candidates'] as List<dynamic>? ?? [])
            .map((c) => RouteCandidate.fromJson(c as Map<String, dynamic>))
            .toList(),
        explanation: json['explanation'] as String?,
      );
}
