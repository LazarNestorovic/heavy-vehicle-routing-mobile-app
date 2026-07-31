import 'package:flutter/material.dart';

import 'screens/entry_router.dart';
import 'screens/login_screen.dart';
import 'services/api_client.dart';
import 'services/auth_storage.dart';
import 'services/route_observer.dart';
import 'theme/nocturne_theme.dart';

/// Lets ApiClient.onUnauthorized navigate to LoginScreen without a
/// BuildContext of its own - a 401 can arrive from any screen's API call.
final navigatorKey = GlobalKey<NavigatorState>();

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final api = ApiClient();
  final authStorage = AuthStorage();
  api.token = await authStorage.loadToken();
  api.username = await authStorage.loadUsername();
  api.driverId = await authStorage.loadDriverId();
  api.role = await authStorage.loadRole();
  api.dispatcherId = await authStorage.loadDispatcherId();
  api.email = await authStorage.loadEmail();
  api.emailVerified = await authStorage.loadEmailVerified();

  // A stale/invalidated token (expired, or "Odjavi sve uređaje" from another
  // device/session) previously just surfaced as a raw "invalid or expired
  // token" ApiException wherever it happened to be hit, with no way back to
  // the login screen short of restarting the app - see documentations/fixes/
  // entry. _handling guards against concurrent in-flight requests all firing
  // this at once (e.g. a screen that fires several API calls on load).
  var handlingUnauthorized = false;
  api.onUnauthorized = () async {
    if (handlingUnauthorized) return;
    handlingUnauthorized = true;
    await clearSession(api);
    navigatorKey.currentState?.pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => LoginScreen(api: api)),
      (route) => false,
    );
    handlingUnauthorized = false;
  };

  runApp(HvrApp(api: api));
}

class HvrApp extends StatelessWidget {
  final ApiClient api;
  const HvrApp({super.key, required this.api});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      navigatorKey: navigatorKey,
      navigatorObservers: [routeObserver],
      title: 'HVR - Vozač',
      theme: buildNocturneTheme(),
      home: api.token != null ? homeScreenFor(api) : LoginScreen(api: api),
    );
  }
}
