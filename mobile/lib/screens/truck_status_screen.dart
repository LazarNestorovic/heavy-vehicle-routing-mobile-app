import 'package:flutter/material.dart';

import '../models/trip.dart';
import '../models/vehicle_hours.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Mock-up's "Truck Status" panel: fuel/service (manually reported - no
/// telematics, see backend/internal/store/vehicle.go) and driving hours
/// (backend/internal/store/trip.go DrivingHours, a simplified stand-in for
/// real AETR tracking). The "rest break recommended" banner reuses the
/// current trip's own next_rest_suggestion_min/rest_stop - it's not a new
/// mechanism, just surfaced here too.
class TruckStatusScreen extends StatefulWidget {
  final ApiClient api;
  final VehicleProfile vehicle;
  final Trip? currentTrip;

  const TruckStatusScreen({super.key, required this.api, required this.vehicle, this.currentTrip});

  @override
  State<TruckStatusScreen> createState() => _TruckStatusScreenState();
}

class _TruckStatusScreenState extends State<TruckStatusScreen> {
  late VehicleProfile _vehicle;
  late Future<VehicleHours> _hoursFuture;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _vehicle = widget.vehicle;
    _hoursFuture = widget.api.getVehicleHours(_vehicle.id!);
  }

  Future<void> _editFuel() async {
    final controller = TextEditingController(text: _vehicle.fuelPercent.toStringAsFixed(0));
    final result = await showDialog<double>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Nivo goriva (%)'),
        content: TextField(
          controller: controller,
          keyboardType: const TextInputType.numberWithOptions(decimal: false),
          decoration: const InputDecoration(suffixText: '%'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Otkaži')),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(double.tryParse(controller.text)),
            child: const Text('Sačuvaj'),
          ),
        ],
      ),
    );
    if (result == null) return;

    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final updated = await widget.api.updateVehicleStatus(
        _vehicle.id!,
        fuelPercent: result.clamp(0, 100),
        nextServiceKm: _vehicle.nextServiceKm,
      );
      if (!mounted) return;
      setState(() => _vehicle = updated);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Greška: $e');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _editNextService() async {
    final controller = TextEditingController(text: _vehicle.nextServiceKm?.toStringAsFixed(0) ?? '');
    final result = await showDialog<double>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Sledeći servis (km)'),
        content: TextField(
          controller: controller,
          keyboardType: const TextInputType.numberWithOptions(decimal: false),
          decoration: const InputDecoration(suffixText: 'km'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('Otkaži')),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(double.tryParse(controller.text)),
            child: const Text('Sačuvaj'),
          ),
        ],
      ),
    );
    if (result == null) return;

    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final updated = await widget.api.updateVehicleStatus(
        _vehicle.id!,
        fuelPercent: _vehicle.fuelPercent,
        nextServiceKm: result,
      );
      if (!mounted) return;
      setState(() => _vehicle = updated);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Greška: $e');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final trip = widget.currentTrip;
    final restStop = trip?.restStop;

    return Scaffold(
      appBar: AppBar(title: const Text('Status vozila')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
            ),
          if (trip?.nextRestSuggestionMin != null)
            Card(
              color: NocturneColors.accent800,
              child: ListTile(
                leading: const Icon(Icons.bedtime, color: NocturneColors.accent300),
                title: const Text('Preporučena pauza'),
                subtitle: Text(
                  restStop != null
                      ? '${restStop.label} (${restStop.amenityLabel})'
                      : 'Nakon ${trip!.nextRestSuggestionMin!.toStringAsFixed(0)} min vožnje',
                ),
              ),
            ),
          const SizedBox(height: 8),
          Card(
            child: ListTile(
              leading: const Icon(Icons.local_gas_station),
              title: const Text('Gorivo'),
              subtitle: Text('${_vehicle.fuelPercent.toStringAsFixed(0)}%'),
              trailing: _saving ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.edit),
              onTap: _saving ? null : _editFuel,
            ),
          ),
          Card(
            child: ListTile(
              leading: const Icon(Icons.build),
              title: const Text('Sledeći servis'),
              subtitle: Text(_vehicle.nextServiceKm != null ? '${_vehicle.nextServiceKm!.toStringAsFixed(0)} km' : 'Nije podešeno'),
              trailing: _saving ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)) : const Icon(Icons.edit),
              onTap: _saving ? null : _editNextService,
            ),
          ),
          const SizedBox(height: 8),
          FutureBuilder<VehicleHours>(
            future: _hoursFuture,
            builder: (context, snapshot) {
              if (snapshot.connectionState == ConnectionState.waiting) {
                return const Padding(
                  padding: EdgeInsets.all(16),
                  child: Center(child: CircularProgressIndicator()),
                );
              }
              if (snapshot.hasError) {
                return Padding(
                  padding: const EdgeInsets.all(8),
                  child: Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)),
                );
              }
              final hours = snapshot.data!;
              return Card(
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Radni sati', style: Theme.of(context).textTheme.titleMedium),
                      const SizedBox(height: 8),
                      Text('Od poslednje pauze: ${hours.sinceLastBreakMin.toStringAsFixed(0)} min'),
                      Text('Vožnja danas: ${hours.drivingTodayMin.toStringAsFixed(0)} min'),
                    ],
                  ),
                ),
              );
            },
          ),
        ],
      ),
    );
  }
}
