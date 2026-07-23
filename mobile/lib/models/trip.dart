import 'rest_stop.dart';

/// Response of POST /api/v1/trips and GET /api/v1/trips/{id} - the persisted trip.
class Trip {
  final int id;
  final int vehicleId;
  final String status;
  final double distanceKm;
  final double durationMin;
  final String shape;
  final double riskScore;
  final String? explanation;
  final double? nextRestSuggestionMin;
  final RestStop? restStop;

  const Trip({
    required this.id,
    required this.vehicleId,
    required this.status,
    required this.distanceKm,
    required this.durationMin,
    required this.shape,
    required this.riskScore,
    this.explanation,
    this.nextRestSuggestionMin,
    this.restStop,
  });

  factory Trip.fromJson(Map<String, dynamic> json) => Trip(
        id: json['id'] as int,
        vehicleId: json['vehicle_id'] as int,
        status: json['status'] as String,
        distanceKm: (json['distance_km'] as num).toDouble(),
        durationMin: (json['duration_min'] as num).toDouble(),
        shape: json['shape'] as String,
        riskScore: (json['risk_score'] as num).toDouble(),
        explanation: json['explanation'] as String?,
        nextRestSuggestionMin: (json['next_rest_suggestion_min'] as num?)?.toDouble(),
        restStop: json['rest_stop'] != null ? RestStop.fromJson(json['rest_stop'] as Map<String, dynamic>) : null,
      );
}
