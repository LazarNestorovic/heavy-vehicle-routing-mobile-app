import 'package:flutter/material.dart';

import 'screens/vehicle_profile_screen.dart';

void main() {
  runApp(const HvrApp());
}

class HvrApp extends StatelessWidget {
  const HvrApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'HVR - Vozač',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
        useMaterial3: true,
      ),
      home: const VehicleProfileScreen(),
    );
  }
}
