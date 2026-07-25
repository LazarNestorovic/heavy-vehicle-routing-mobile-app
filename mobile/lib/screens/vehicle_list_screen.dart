import 'package:flutter/material.dart';

import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/auth_storage.dart';
import 'dispatcher_requests_screen.dart';
import 'login_screen.dart';
import 'preferences_screen.dart';
import 'route_request_screen.dart';
import 'vehicle_profile_screen.dart';

/// Hub screen after login (SPECIFIKACIJA.md 3.9 + Faza 2/6 of the driver-preference
/// plan): lists the driver's own vehicles (1:N, see documentations/features/
/// 2026-07-21-driver-preference-scoring.md), pick one to start planning a route,
/// or add a new one.
class VehicleListScreen extends StatefulWidget {
  final ApiClient api;
  const VehicleListScreen({super.key, required this.api});

  @override
  State<VehicleListScreen> createState() => _VehicleListScreenState();
}

class _VehicleListScreenState extends State<VehicleListScreen> {
  late Future<List<VehicleProfile>> _vehiclesFuture;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _vehiclesFuture = widget.api.listVehicles();
    });
  }

  Future<void> _addVehicle() async {
    final created = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => VehicleProfileScreen(api: widget.api)),
    );
    if (created == true) _reload();
  }

  Future<void> _logout() async {
    await AuthStorage().clear();
    widget.api.token = null;
    widget.api.driverId = null;
    widget.api.username = null;
    widget.api.role = null;
    widget.api.dispatcherId = null;
    if (!mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => LoginScreen(api: widget.api)),
      (route) => false,
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Moja vozila'),
        actions: [
          IconButton(
            icon: const Icon(Icons.tune),
            tooltip: 'Preference',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => PreferencesScreen(api: widget.api)),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.business),
            tooltip: 'Zahtevi dispečera',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => DispatcherRequestsScreen(api: widget.api)),
            ),
          ),
          IconButton(icon: const Icon(Icons.logout), tooltip: 'Odjava', onPressed: _logout),
        ],
      ),
      body: FutureBuilder<List<VehicleProfile>>(
        future: _vehiclesFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(child: Text('Greška: ${snapshot.error}'));
          }
          final vehicles = snapshot.data ?? [];
          if (vehicles.isEmpty) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Text('Nemaš još sačuvano vozilo.'),
                  const SizedBox(height: 12),
                  FilledButton(onPressed: _addVehicle, child: const Text('Dodaj vozilo')),
                ],
              ),
            );
          }
          return ListView.builder(
            itemCount: vehicles.length,
            itemBuilder: (context, i) {
              final v = vehicles[i];
              return ListTile(
                leading: const Icon(Icons.local_shipping),
                title: Text('${v.heightM}m / ${v.widthM}m / ${v.lengthM}m'),
                subtitle: Text('${v.weightKg.toStringAsFixed(0)}kg${v.hazmat ? " · hazmat" : ""}'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => RouteRequestScreen(api: widget.api, vehicle: v)),
                ),
              );
            },
          );
        },
      ),
      floatingActionButton: FloatingActionButton(onPressed: _addVehicle, child: const Icon(Icons.add)),
    );
  }
}
