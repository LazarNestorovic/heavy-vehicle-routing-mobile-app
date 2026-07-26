import 'rest_stop.dart';
import 'route_candidate.dart';

/// Response of POST /api/v1/trips and GET /api/v1/trips/{id} - the persisted trip.
class Trip {
  final int id;
  final int driverId;
  final String? driverUsername;
  final int vehicleId;
  final String status;
  final double distanceKm;
  final double durationMin;
  final String shape;
  final double riskScore;
  final List<RouteCandidate> candidates;
  final String? explanation;
  final double? nextRestSuggestionMin;
  final RestStop? restStop;
  final String? cargoDescription;
  final double? cargoWeightKg;
  final String? cargoTempRange;
  final String? pickupLocation;
  final String? dropoffLocation;

  const Trip({
    required this.id,
    required this.driverId,
    this.driverUsername,
    required this.vehicleId,
    required this.status,
    required this.distanceKm,
    required this.durationMin,
    required this.shape,
    required this.riskScore,
    this.candidates = const [],
    this.explanation,
    this.nextRestSuggestionMin,
    this.restStop,
    this.cargoDescription,
    this.cargoWeightKg,
    this.cargoTempRange,
    this.pickupLocation,
    this.dropoffLocation,
  });

  factory Trip.fromJson(Map<String, dynamic> json) => Trip(
        id: json['id'] as int,
        driverId: json['driver_id'] as int,
        driverUsername: json['driver_username'] as String?,
        vehicleId: json['vehicle_id'] as int,
        status: json['status'] as String,
        distanceKm: (json['distance_km'] as num).toDouble(),
        durationMin: (json['duration_min'] as num).toDouble(),
        shape: json['shape'] as String,
        riskScore: (json['risk_score'] as num).toDouble(),
        candidates: (json['candidates'] as List<dynamic>? ?? [])
            .map((c) => RouteCandidate.fromJson(c as Map<String, dynamic>))
            .toList(),
        explanation: json['explanation'] as String?,
        nextRestSuggestionMin: (json['next_rest_suggestion_min'] as num?)?.toDouble(),
        restStop: json['rest_stop'] != null ? RestStop.fromJson(json['rest_stop'] as Map<String, dynamic>) : null,
        cargoDescription: json['cargo_description'] as String?,
        cargoWeightKg: (json['cargo_weight_kg'] as num?)?.toDouble(),
        cargoTempRange: json['cargo_temp_range'] as String?,
        pickupLocation: json['pickup_location'] as String?,
        dropoffLocation: json['dropoff_location'] as String?,
      );
}
