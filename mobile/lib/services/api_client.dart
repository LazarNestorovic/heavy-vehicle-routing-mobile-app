import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:latlong2/latlong.dart';

import '../config.dart';
import '../models/route_result.dart';
import '../models/trip.dart';
import '../models/vehicle_profile.dart';

class ApiException implements Exception {
  final String message;
  ApiException(this.message);
  @override
  String toString() => message;
}

/// Thin wrapper around the backend REST API (see backend/internal/httpapi).
class ApiClient {
  final http.Client _client;

  ApiClient({http.Client? client}) : _client = client ?? http.Client();

  Future<VehicleProfile> createVehicle(VehicleProfile profile) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/vehicles'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(profile.toJson()),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return VehicleProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<RouteResult> previewRoute({
    required LatLng origin,
    required LatLng destination,
    required VehicleProfile vehicle,
  }) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/routes'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'origin': {'lat': origin.latitude, 'lon': origin.longitude},
        'destination': {'lat': destination.latitude, 'lon': destination.longitude},
        'vehicle': vehicle.toJson(),
      }),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return RouteResult.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<Trip> createTrip({
    required int vehicleId,
    required LatLng origin,
    required LatLng destination,
  }) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/trips'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'vehicle_id': vehicleId,
        'origin': {'lat': origin.latitude, 'lon': origin.longitude},
        'destination': {'lat': destination.latitude, 'lon': destination.longitude},
      }),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<Trip> getTrip(int id) async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/trips/$id'));
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  String _errorMessage(http.Response resp) {
    try {
      final body = jsonDecode(resp.body) as Map<String, dynamic>;
      return body['error'] as String? ?? 'HTTP ${resp.statusCode}';
    } catch (_) {
      return 'HTTP ${resp.statusCode}';
    }
  }
}
