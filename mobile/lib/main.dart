import 'package:flutter/material.dart';

import 'screens/login_screen.dart';
import 'screens/vehicle_list_screen.dart';
import 'services/api_client.dart';
import 'services/auth_storage.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final api = ApiClient();
  api.token = await AuthStorage().loadToken();

  runApp(HvrApp(api: api));
}

class HvrApp extends StatelessWidget {
  final ApiClient api;
  const HvrApp({super.key, required this.api});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'HVR - Vozač',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
        useMaterial3: true,
      ),
      home: api.token != null ? VehicleListScreen(api: api) : LoginScreen(api: api),
    );
  }
}
