import 'package:flutter/material.dart';

import 'screens/entry_router.dart';
import 'screens/login_screen.dart';
import 'services/api_client.dart';
import 'services/auth_storage.dart';
import 'theme/nocturne_theme.dart';

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

  runApp(HvrApp(api: api));
}

class HvrApp extends StatelessWidget {
  final ApiClient api;
  const HvrApp({super.key, required this.api});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'HVR - Vozač',
      theme: buildNocturneTheme(),
      home: api.token != null ? homeScreenFor(api) : LoginScreen(api: api),
    );
  }
}
