import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:latlong2/latlong.dart';

import '../config.dart';
import '../models/auth_result.dart';
import '../models/driver_preferences.dart';
import '../models/favorite_stop.dart';
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
/// Every endpoint except /auth/register and /auth/login requires a token
/// (backend/internal/httpapi/middleware.go RequireAuth) - set [token] after
/// login/register (or after loading one from AuthStorage) before calling
/// anything else.
class ApiClient {
  final http.Client _client;
  String? token;

  ApiClient({http.Client? client, this.token}) : _client = client ?? http.Client();

  Map<String, String> get _authHeaders => {
        'Content-Type': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
      };

  Future<AuthResult> register(String username, String password) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/auth/register'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'username': username, 'password': password}),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return AuthResult.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<AuthResult> login(String username, String password) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/auth/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'username': username, 'password': password}),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return AuthResult.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<VehicleProfile> createVehicle(VehicleProfile profile) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/vehicles'),
      headers: _authHeaders,
      body: jsonEncode(profile.toJson()),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return VehicleProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<List<VehicleProfile>> listVehicles() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/vehicles'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((v) => VehicleProfile.fromJson(v as Map<String, dynamic>)).toList();
  }

  Future<RouteResult> previewRoute({
    required LatLng origin,
    required LatLng destination,
    required VehicleProfile vehicle,
  }) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/routes'),
      headers: _authHeaders,
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
      headers: _authHeaders,
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
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/trips/$id'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<DriverPreferences> getPreferences() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/preferences'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return DriverPreferences.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<DriverPreferences> updatePreferences(DriverPreferences prefs) async {
    final resp = await _client.put(
      Uri.parse('$apiBaseUrl/api/v1/preferences'),
      headers: _authHeaders,
      body: jsonEncode(prefs.toJson()),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return DriverPreferences.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<List<FavoriteStop>> listFavoriteStops() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/favorite-stops'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((s) => FavoriteStop.fromJson(s as Map<String, dynamic>)).toList();
  }

  Future<FavoriteStop> createFavoriteStop({required double lat, required double lon, required String name}) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/favorite-stops'),
      headers: _authHeaders,
      body: jsonEncode({'lat': lat, 'lon': lon, 'name': name}),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return FavoriteStop.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<void> deleteFavoriteStop(int id) async {
    final resp = await _client.delete(Uri.parse('$apiBaseUrl/api/v1/favorite-stops/$id'), headers: _authHeaders);
    if (resp.statusCode != 204) {
      throw ApiException(_errorMessage(resp));
    }
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
