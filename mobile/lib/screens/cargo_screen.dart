import 'package:flutter/material.dart';

import '../models/trip.dart';

/// Mock-up's "Cargo" panel: read-only view of the active trip's cargo fields
/// (entered on RouteRequestScreen before departure - there's no separate
/// PATCH endpoint for cargo, it's fixed at trip creation).
class CargoScreen extends StatelessWidget {
  final Trip trip;
  const CargoScreen({super.key, required this.trip});

  @override
  Widget build(BuildContext context) {
    final hasCargo = trip.cargoDescription != null ||
        trip.cargoWeightKg != null ||
        trip.cargoTempRange != null ||
        trip.pickupLocation != null ||
        trip.dropoffLocation != null;

    return Scaffold(
      appBar: AppBar(title: const Text('Tovar')),
      body: !hasCargo
          ? const Center(child: Text('Nema unetih podataka o tovaru za ovo putovanje.'))
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                if (trip.cargoDescription != null)
                  _InfoTile(icon: Icons.inventory_2, title: 'Opis tovara', value: trip.cargoDescription!),
                if (trip.cargoWeightKg != null)
                  _InfoTile(icon: Icons.scale, title: 'Težina tovara', value: '${trip.cargoWeightKg!.toStringAsFixed(0)} kg'),
                if (trip.cargoTempRange != null)
                  _InfoTile(icon: Icons.thermostat, title: 'Temperaturni opseg', value: trip.cargoTempRange!),
                if (trip.pickupLocation != null)
                  _InfoTile(icon: Icons.trip_origin, title: 'Preuzimanje', value: trip.pickupLocation!),
                if (trip.dropoffLocation != null)
                  _InfoTile(icon: Icons.flag, title: 'Isporuka', value: trip.dropoffLocation!),
              ],
            ),
    );
  }
}

class _InfoTile extends StatelessWidget {
  final IconData icon;
  final String title;
  final String value;
  const _InfoTile({required this.icon, required this.title, required this.value});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        leading: Icon(icon),
        title: Text(title),
        subtitle: Text(value),
      ),
    );
  }
}
