// Backend base URL.
//
// - Android emulator: host machine's localhost is reachable at 10.0.2.2 (this
//   is the default below).
// - iOS simulator: localhost works directly - use http://127.0.0.1:8080 and
//   ws://127.0.0.1:8080.
// - Physical device: use your machine's LAN IP (e.g. http://192.168.1.23:8080),
//   the device and the backend must be reachable on the same network.
//
// See documentations/guides/run-flutter-app.md for how to switch this.
const String apiBaseUrl = 'http://192.168.1.9:8080';
const String wsBaseUrl = 'ws://192.168.1.9:8080';

// The "Web application" OAuth client ID from documentations/guides/
// google-maps-setup.md step 7 - passed to google_sign_in as serverClientId
// so the ID token's `aud` claim matches what the backend checks
// (GOOGLE_CLIENT_ID env var, see internal/auth/google.go). Empty until that
// step is done; Google sign-in just won't work until it's filled in (the
// backend independently reports itself as "not configured" either way - see
// documentations/guides/google-maps-setup.md).
const String googleServerClientId = '566978705997-1fee96vt3dkceukfbsitmqmjkfbkc1tk.apps.googleusercontent.com';
