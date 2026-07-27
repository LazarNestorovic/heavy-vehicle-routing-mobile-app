import 'dart:convert';

import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:http/http.dart' as http;

import '../config.dart';
import '../models/account_status.dart';
import '../models/auth_result.dart';
import '../models/chat_message.dart';
import '../models/dispatcher_request.dart';
import '../models/driver.dart';
import '../models/driver_preferences.dart';
import '../models/favorite_stop.dart';
import '../models/geocode_result.dart';
import '../models/route_result.dart';
import '../models/trip.dart';
import '../models/trip_event.dart';
import '../models/vehicle_hours.dart';
import '../models/vehicle_profile.dart';
import 'auth_storage.dart';

class ApiException implements Exception {
  final String message;
  ApiException(this.message);
  @override
  String toString() => message;
}

/// Wraps the real http.Client to detect a 401 response to an AUTHENTICATED
/// request (one that carried an Authorization header) and report it via
/// [onUnauthorized] - see ApiClient.onUnauthorized. A 401 from an
/// unauthenticated call (wrong password on /auth/login, which never sends
/// Authorization) must NOT trigger this - there's no session to invalidate
/// in that case.
class _AuthAwareClient extends http.BaseClient {
  final http.Client _inner;
  final void Function() _onUnauthorized;
  _AuthAwareClient(this._inner, this._onUnauthorized);

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final response = await _inner.send(request);
    if (response.statusCode == 401 && request.headers.containsKey('Authorization')) {
      _onUnauthorized();
    }
    return response;
  }

  @override
  void close() => _inner.close();
}

/// Thin wrapper around the backend REST API (see backend/internal/httpapi).
/// Every endpoint except /auth/register and /auth/login requires a token
/// (backend/internal/httpapi/middleware.go RequireAuth) - set [token] after
/// login/register (or after loading one from AuthStorage) before calling
/// anything else.
class ApiClient {
  late final http.Client _client;
  String? token;
  int? driverId;
  String? username;
  String? role;
  int? dispatcherId;
  String? email;
  bool emailVerified;

  /// Fires whenever an authenticated request gets a 401 back - the token was
  /// rejected (expired, or invalidated by "Odjavi sve uređaje" on another
  /// device/session, see documentations/fixes/ entry). Set once at app
  /// startup (main.dart) to force the user back to the login screen - without
  /// this, a stale session just surfaced as a raw ApiException on whatever
  /// screen happened to be open, with no way to recover except manually
  /// restarting the app.
  void Function()? onUnauthorized;

  ApiClient({http.Client? client, this.token, this.emailVerified = false}) {
    _client = _AuthAwareClient(client ?? http.Client(), () => onUnauthorized?.call());
  }

  Map<String, String> get _authHeaders => {
        'Content-Type': 'application/json',
        if (token != null) 'Authorization': 'Bearer $token',
      };

  // email is required (see backend/internal/httpapi/auth.go validate()) -
  // registration always sends a verification link.
  Future<AuthResult> register(String username, String password, {String role = 'driver', required String email}) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/auth/register'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'username': username,
        'password': password,
        'role': role,
        'email': email,
      }),
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

  /// Exchanges a Google ID token (see services/google_auth.dart) for a normal
  /// session, same as [login]/[register]. [role] is only used if this Google
  /// account has no matching driver row yet (first-time sign-in).
  Future<AuthResult> signInWithGoogle(String idToken, {String role = 'driver'}) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/auth/google'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'id_token': idToken, 'role': role}),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return AuthResult.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<AccountStatus> fetchAccountStatus() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/auth/me'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return AccountStatus.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Refreshes email/emailVerified/role/dispatcherId from the server and
  /// persists the update - picks up an email verification that happened
  /// outside the app (clicking the emailed link in a browser) without
  /// requiring a full re-login. See widgets/email_verification_banner.dart,
  /// which calls this on app resume.
  Future<void> refreshAccountStatus() async {
    final status = await fetchAccountStatus();
    email = status.email;
    emailVerified = status.emailVerified;
    role = status.role;
    dispatcherId = status.dispatcherId;
    await AuthStorage().saveEmailVerified(emailVerified);
    await AuthStorage().saveDispatcherId(dispatcherId);
  }

  /// Ends the caller's dispatcher relationship (driver-initiated - see
  /// documentations/fixes/2026-07-26-*.md). Fleet vehicles become
  /// inaccessible immediately; the driver's own personal vehicles and past
  /// trips are unaffected.
  Future<void> leaveDispatcher() async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/driver/leave-dispatcher'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    dispatcherId = null;
    await AuthStorage().saveDispatcherId(null);
  }

  Future<void> resendVerificationEmail() async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/auth/resend-verification'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
  }

  /// Always succeeds regardless of whether [email] matches an account (see
  /// backend/internal/httpapi/auth.go handleForgotPassword - doesn't leak
  /// which addresses are registered). The actual reset happens on a web page
  /// opened from the emailed link, not in the app.
  Future<void> forgotPassword(String email) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/auth/forgot-password'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'email': email}),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
  }

  /// Invalidates every token previously issued to this account (see
  /// Driver.TokenVersion) - including the one this call itself just used, so
  /// the caller should treat this the same as a local logout afterward.
  Future<void> logoutAll() async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/auth/logout-all'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
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

  Future<VehicleProfile> getVehicle(int id) async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/vehicles/$id'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return VehicleProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<VehicleProfile> updateVehicle(int id, VehicleProfile profile) async {
    final resp = await _client.put(
      Uri.parse('$apiBaseUrl/api/v1/vehicles/$id'),
      headers: _authHeaders,
      body: jsonEncode(profile.toJson()),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return VehicleProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Throws ApiException with a 409 message if the vehicle still has trips
  /// associated with it (see backend/internal/store/vehicle.go ErrVehicleInUse).
  Future<void> deleteVehicle(int id) async {
    final resp = await _client.delete(Uri.parse('$apiBaseUrl/api/v1/vehicles/$id'), headers: _authHeaders);
    if (resp.statusCode != 204) {
      throw ApiException(_errorMessage(resp));
    }
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

  /// Creates a trip. [driverId] is required when the caller is a dispatcher
  /// (who this trip is assigned to) - it saves as status "offered" and the
  /// driver starts it themselves via [startTrip]. Omit for self-service
  /// (independent driver) creation, which starts immediately as before.
  Future<Trip> createTrip({
    required int vehicleId,
    required LatLng origin,
    required LatLng destination,
    String? cargoDescription,
    double? cargoWeightKg,
    String? cargoTempRange,
    String? pickupLocation,
    String? dropoffLocation,
    int? driverId,
  }) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/trips'),
      headers: _authHeaders,
      body: jsonEncode({
        'vehicle_id': vehicleId,
        'origin': {'lat': origin.latitude, 'lon': origin.longitude},
        'destination': {'lat': destination.latitude, 'lon': destination.longitude},
        if (cargoDescription != null) 'cargo_description': cargoDescription,
        if (cargoWeightKg != null) 'cargo_weight_kg': cargoWeightKg,
        if (cargoTempRange != null) 'cargo_temp_range': cargoTempRange,
        if (pickupLocation != null) 'pickup_location': pickupLocation,
        if (dropoffLocation != null) 'dropoff_location': dropoffLocation,
        if (driverId != null) 'driver_id': driverId,
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

  /// Lists the caller's own trips (self-service and assigned, for a driver;
  /// assigned-by-them, for a dispatcher). Optional status filter (e.g.
  /// "offered", "in_progress").
  Future<List<Trip>> listMyTrips({String? status}) async {
    final uri = Uri.parse('$apiBaseUrl/api/v1/trips').replace(queryParameters: status != null ? {'status': status} : null);
    final resp = await _client.get(uri, headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((t) => Trip.fromJson(t as Map<String, dynamic>)).toList();
  }

  /// Accepts a dispatcher-assigned "offered" trip -> "accepted" - driver-only.
  /// The driver has reviewed it but hasn't departed yet (see [startTrip]).
  Future<Trip> acceptTrip(int id) async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/trips/$id/accept'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Declines a dispatcher-assigned "offered" trip -> "rejected" - driver-only.
  Future<Trip> rejectTrip(int id) async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/trips/$id/reject'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Starts an "accepted" trip (-> active) - driver-only.
  Future<Trip> startTrip(int id) async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/trips/$id/start'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Reports a real GPS fix for an active trip (driver-only) - switches every
  /// WS watcher of this trip (this screen, and any dispatcher watching the
  /// fleet map) from the simulated route playback to live relaying. See
  /// services/location_service.dart and backend/internal/ws/gateway.go.
  Future<void> reportPosition(int tripId, double lat, double lon) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/trips/$tripId/position'),
      headers: _authHeaders,
      body: jsonEncode({'lat': lat, 'lon': lon}),
    );
    if (resp.statusCode != 204) {
      throw ApiException(_errorMessage(resp));
    }
  }

  /// Explicitly confirms arrival (driver-only) - needed for live-GPS trips,
  /// which have no reliable auto-arrival signal the way the simulated WS
  /// playback's progress_fraction=1 does.
  Future<Trip> completeTrip(int tripId) async {
    final resp = await _client.post(Uri.parse('$apiBaseUrl/api/v1/trips/$tripId/complete'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return Trip.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Address search (see widgets/address_search_field.dart) - GET /api/v1/geocode,
  /// a thin authenticated proxy to Nominatim (internal/geocode).
  Future<List<GeocodeResult>> geocodeSearch(String query) async {
    final uri = Uri.parse('$apiBaseUrl/api/v1/geocode').replace(queryParameters: {'q': query, 'limit': '5'});
    final resp = await _client.get(uri, headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((r) => GeocodeResult.fromJson(r as Map<String, dynamic>)).toList();
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

  Future<List<Driver>> listDrivers() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/drivers'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((d) => Driver.fromJson(d as Map<String, dynamic>)).toList();
  }

  Future<VehicleProfile> updateVehicleStatus(int vehicleId, {required double fuelPercent, double? nextServiceKm}) async {
    final resp = await _client.patch(
      Uri.parse('$apiBaseUrl/api/v1/vehicles/$vehicleId/status'),
      headers: _authHeaders,
      body: jsonEncode({
        'fuel_percent': fuelPercent,
        if (nextServiceKm != null) 'next_service_km': nextServiceKm,
      }),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return VehicleProfile.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<VehicleHours> getVehicleHours(int vehicleId) async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/vehicles/$vehicleId/hours'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    return VehicleHours.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<List<TripEvent>> listTripEvents(int tripId) async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/trips/$tripId/events'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((e) => TripEvent.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<ChatConversation>> listChats() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/chats'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((c) => ChatConversation.fromJson(c as Map<String, dynamic>)).toList();
  }

  Future<List<ChatMessage>> getChatMessages(int counterpartId) async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/chats/$counterpartId/messages'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((m) => ChatMessage.fromJson(m as Map<String, dynamic>)).toList();
  }

  Future<ChatMessage> sendChatMessage(int counterpartId, String body) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/chats/$counterpartId/messages'),
      headers: _authHeaders,
      body: jsonEncode({'body': body}),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return ChatMessage.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  /// Fleet vehicles + that managed driver's own personal vehicles - the
  /// dispatcher create-trip picker's vehicle list (see documentations/
  /// features/ entry). Dispatcher-only; driverId must be one of theirs.
  Future<List<VehicleProfile>> listDriverVehicles(int driverId) async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/dispatcher/drivers/$driverId/vehicles'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((v) => VehicleProfile.fromJson(v as Map<String, dynamic>)).toList();
  }

  Future<List<Driver>> listManagedDrivers() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/dispatcher/drivers'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((d) => Driver.fromJson(d as Map<String, dynamic>)).toList();
  }

  Future<List<Driver>> listAvailableDrivers() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/dispatcher/available-drivers'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((d) => Driver.fromJson(d as Map<String, dynamic>)).toList();
  }

  Future<DispatcherRequest> sendDispatcherRequest(int driverId) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/dispatcher/requests'),
      headers: _authHeaders,
      body: jsonEncode({'driver_id': driverId}),
    );
    if (resp.statusCode != 201) {
      throw ApiException(_errorMessage(resp));
    }
    return DispatcherRequest.fromJson(jsonDecode(resp.body) as Map<String, dynamic>);
  }

  Future<List<DispatcherRequest>> listSentDispatcherRequests() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/dispatcher/requests'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((r) => DispatcherRequest.fromJson(r as Map<String, dynamic>)).toList();
  }

  Future<List<DispatcherRequest>> listDriverRequests() async {
    final resp = await _client.get(Uri.parse('$apiBaseUrl/api/v1/driver/requests'), headers: _authHeaders);
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final list = jsonDecode(resp.body) as List<dynamic>;
    return list.map((r) => DispatcherRequest.fromJson(r as Map<String, dynamic>)).toList();
  }

  /// Returns the updated dispatcher_id (non-null when [approve] was true) so
  /// the caller can update local session state without re-logging in.
  Future<int?> respondDispatcherRequest(int requestId, bool approve) async {
    final resp = await _client.post(
      Uri.parse('$apiBaseUrl/api/v1/driver/requests/$requestId/respond'),
      headers: _authHeaders,
      body: jsonEncode({'approve': approve}),
    );
    if (resp.statusCode != 200) {
      throw ApiException(_errorMessage(resp));
    }
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return body['dispatcher_id'] as int?;
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
