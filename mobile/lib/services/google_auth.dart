import 'package:google_sign_in/google_sign_in.dart';

import '../config.dart';

/// Thin wrapper around google_sign_in v7's singleton API - see
/// documentations/guides/google-maps-setup.md for the Google Cloud OAuth
/// client setup this depends on (serverClientId is the "Web application"
/// client ID from that guide's step 7 - it must match the backend's
/// GOOGLE_CLIENT_ID so the ID token's `aud` claim verifies).
class GoogleAuthService {
  bool _initialized = false;

  Future<void> _ensureInitialized() async {
    if (_initialized) return;
    await GoogleSignIn.instance.initialize(serverClientId: googleServerClientId);
    _initialized = true;
  }

  /// Runs the interactive Google sign-in flow and returns the ID token to
  /// send to POST /api/v1/auth/google - or null if the user canceled (not an
  /// error, just declined to continue).
  Future<String?> signIn() async {
    await _ensureInitialized();
    try {
      final account = await GoogleSignIn.instance.authenticate();
      return account.authentication.idToken;
    } on GoogleSignInException catch (e) {
      if (e.code == GoogleSignInExceptionCode.canceled) return null;
      rethrow;
    }
  }
}
