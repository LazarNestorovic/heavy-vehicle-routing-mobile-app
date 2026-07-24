import 'package:shared_preferences/shared_preferences.dart';

/// Persists the JWT locally, same pattern as vehicle storage used to. No
/// server-side revocation (see backend/internal/auth) - "logout" is just
/// clearing this.
class AuthStorage {
  static const _keyToken = 'auth_token';
  static const _keyDriverId = 'auth_driver_id';

  Future<String?> loadToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_keyToken);
  }

  Future<void> save(String token, int driverId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyToken, token);
    await prefs.setInt(_keyDriverId, driverId);
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyToken);
    await prefs.remove(_keyDriverId);
  }
}
