class RouteCandidate {
  final double distanceKm;
  final double durationMin;
  final double riskScore;
  final int maneuverCount;
  final double highwayRatio;
  final bool hasFerry;
  final bool hasToll;
  final bool chosen;
  final String shape;

  const RouteCandidate({
    required this.distanceKm,
    required this.durationMin,
    required this.riskScore,
    required this.maneuverCount,
    required this.highwayRatio,
    required this.hasFerry,
    required this.hasToll,
    required this.chosen,
    required this.shape,
  });

  factory RouteCandidate.fromJson(Map<String, dynamic> json) => RouteCandidate(
        distanceKm: (json['distance_km'] as num).toDouble(),
        durationMin: (json['duration_min'] as num).toDouble(),
        riskScore: (json['risk_score'] as num).toDouble(),
        maneuverCount: json['maneuver_count'] as int,
        highwayRatio: (json['highway_ratio'] as num).toDouble(),
        hasFerry: json['has_ferry'] as bool,
        hasToll: json['has_toll'] as bool,
        chosen: json['chosen'] as bool,
        shape: json['shape'] as String? ?? '',
      );
}
