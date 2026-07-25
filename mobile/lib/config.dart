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
const String apiBaseUrl = 'http://192.168.1.13:8080';
const String wsBaseUrl = 'ws://192.168.1.13:8080';
