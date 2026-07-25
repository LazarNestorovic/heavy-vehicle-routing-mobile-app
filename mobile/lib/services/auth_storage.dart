import 'package:shared_preferences/shared_preferences.dart';

/// Persists the JWT locally, same pattern as vehicle storage used to. No
/// server-side revocation (see backend/internal/auth) - "logout" is just
/// clearing this.
class AuthStorage {
  static const _keyToken = 'auth_token';
  static const _keyDriverId = 'auth_driver_id';
  static const _keyUsername = 'auth_username';
  static const _keyRole = 'auth_role';
  static const _keyDispatcherId = 'auth_dispatcher_id';

  Future<String?> loadToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_keyToken);
  }

  /// There's no GET /api/v1/me - the username is only known at login/register
  /// time (the driver typed it), so it's persisted here alongside the token
  /// rather than re-fetched.
  Future<String?> loadUsername() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_keyUsername);
  }

  Future<int?> loadDriverId() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_keyDriverId);
  }

  Future<String?> loadRole() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_keyRole);
  }

  Future<int?> loadDispatcherId() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt(_keyDispatcherId);
  }

  Future<void> save(String token, int driverId, String username, String role, int? dispatcherId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyToken, token);
    await prefs.setInt(_keyDriverId, driverId);
    await prefs.setString(_keyUsername, username);
    await prefs.setString(_keyRole, role);
    await saveDispatcherId(dispatcherId);
  }

  /// Updates just the dispatcher link, without touching the rest of the saved
  /// session - used after a driver approves a DispatcherRequest mid-session
  /// (see dispatcher_requests_screen.dart), so they don't need to re-login.
  Future<void> saveDispatcherId(int? dispatcherId) async {
    final prefs = await SharedPreferences.getInstance();
    if (dispatcherId == null) {
      await prefs.remove(_keyDispatcherId);
    } else {
      await prefs.setInt(_keyDispatcherId, dispatcherId);
    }
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyToken);
    await prefs.remove(_keyDriverId);
    await prefs.remove(_keyUsername);
    await prefs.remove(_keyRole);
    await prefs.remove(_keyDispatcherId);
  }
}
