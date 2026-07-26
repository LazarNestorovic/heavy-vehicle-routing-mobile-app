import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../models/driver.dart';
import '../models/route_result.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/polyline.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/address_search_field.dart';

/// Dispatcher's route-planning screen for a specific managed driver - the
/// dispatcher counterpart of RouteRequestScreen (see documentations/features/
/// entry for the dispatcher/driver roles feature). Vehicle picker offers both
/// the dispatcher's own fleet AND that driver's personal vehicles (GET
/// /api/v1/dispatcher/drivers/{id}/vehicles), grouped by label.
class DispatcherCreateTripScreen extends StatefulWidget {
  final ApiClient api;
  final Driver driver;
  const DispatcherCreateTripScreen({super.key, required this.api, required this.driver});

  @override
  State<DispatcherCreateTripScreen> createState() => _DispatcherCreateTripScreenState();
}

class _DispatcherCreateTripScreenState extends State<DispatcherCreateTripScreen> {
  late Future<List<VehicleProfile>> _vehiclesFuture;
  VehicleProfile? _selectedVehicle;

  GoogleMapController? _mapController;
  LatLng? _origin;
  LatLng? _destination;
  RouteResult? _routeResult;
  List<LatLng> _routePoints = [];

  final _cargoDescriptionCtrl = TextEditingController();
  final _cargoWeightCtrl = TextEditingController();
  final _cargoTempRangeCtrl = TextEditingController();
  final _pickupLocationCtrl = TextEditingController();
  final _dropoffLocationCtrl = TextEditingController();

  bool _loading = false;
  String? _error;

  static const _serbiaCenter = LatLng(44.5, 20.5);

  // Not-chosen alternatives from the last preview, drawn thin/grey underneath
  // the chosen route (see documentations/features/ entry) - purely visual.
  Set<Polyline> get _alternatePolylines {
    final candidates = _routeResult?.candidates ?? const [];
    return {
      for (final (i, c) in candidates.indexed)
        if (!c.chosen)
          Polyline(
            polylineId: PolylineId('alt_$i'),
            points: decodePolyline6(c.shape),
            width: 2,
            color: Colors.grey.withValues(alpha: 0.6),
            zIndex: 0,
          ),
    };
  }

  @override
  void initState() {
    super.initState();
    _vehiclesFuture = widget.api.listDriverVehicles(widget.driver.id);
  }

  @override
  void dispose() {
    _cargoDescriptionCtrl.dispose();
    _cargoWeightCtrl.dispose();
    _cargoTempRangeCtrl.dispose();
    _pickupLocationCtrl.dispose();
    _dropoffLocationCtrl.dispose();
    super.dispose();
  }

  void _handleTap(LatLng point) {
    setState(() {
      _routeResult = null;
      _routePoints = [];
      _error = null;
      if (_origin == null) {
        _origin = point;
      } else if (_destination == null) {
        _destination = point;
      } else {
        _origin = point;
        _destination = null;
      }
    });
  }

  // Same origin/destination sequencing as _handleTap, but from an address
  // search result instead of a map tap (see widgets/address_search_field.dart).
  void _handleSearchSelect(LatLng point) {
    _handleTap(point);
    _mapController?.animateCamera(CameraUpdate.newLatLng(point));
  }

  Future<void> _previewRoute() async {
    if (_origin == null || _destination == null || _selectedVehicle == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.api.previewRoute(origin: _origin!, destination: _destination!, vehicle: _selectedVehicle!);
      setState(() {
        _routeResult = result;
        _routePoints = decodePolyline6(result.shape);
      });
    } on ApiException catch (e) {
      setState(() => _error = 'Greška: ${e.message}');
    } catch (e) {
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _assignTrip() async {
    if (_origin == null || _destination == null || _selectedVehicle?.id == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      await widget.api.createTrip(
        vehicleId: _selectedVehicle!.id!,
        origin: _origin!,
        destination: _destination!,
        cargoDescription: _cargoDescriptionCtrl.text.trim().isEmpty ? null : _cargoDescriptionCtrl.text.trim(),
        cargoWeightKg: double.tryParse(_cargoWeightCtrl.text.trim()),
        cargoTempRange: _cargoTempRangeCtrl.text.trim().isEmpty ? null : _cargoTempRangeCtrl.text.trim(),
        pickupLocation: _pickupLocationCtrl.text.trim().isEmpty ? null : _pickupLocationCtrl.text.trim(),
        dropoffLocation: _dropoffLocationCtrl.text.trim().isEmpty ? null : _dropoffLocationCtrl.text.trim(),
        driverId: widget.driver.id,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Tura ponuđena vozaču ${widget.driver.username}.')),
      );
      Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      setState(() => _error = 'Greška: ${e.message}');
    } catch (e) {
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final canAssign = _origin != null && _destination != null && _selectedVehicle != null && !_loading;
    return Scaffold(
      appBar: AppBar(title: Text('Nova tura — ${widget.driver.username}')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            child: FutureBuilder<List<VehicleProfile>>(
              future: _vehiclesFuture,
              builder: (context, snapshot) {
                final vehicles = snapshot.data ?? [];
                if (snapshot.connectionState == ConnectionState.waiting) {
                  return const LinearProgressIndicator();
                }
                if (vehicles.isEmpty) {
                  return const Text('Nema vozila u floti. Dodaj vozilo pre kreiranja ture.');
                }
                return DropdownButtonFormField<VehicleProfile>(
                  initialValue: _selectedVehicle,
                  decoration: const InputDecoration(labelText: 'Vozilo', border: OutlineInputBorder()),
                  items: vehicles
                      .map((v) => DropdownMenuItem(
                            value: v,
                            child: Text(
                              '${v.isFleet ? "Flota" : widget.driver.username} · '
                              '${v.heightM}m/${v.widthM}m/${v.lengthM}m · ${v.weightKg.toStringAsFixed(0)}kg',
                            ),
                          ))
                      .toList(),
                  onChanged: (v) => setState(() => _selectedVehicle = v),
                );
              },
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: AddressSearchField(api: widget.api, onSelected: (r) => _handleSearchSelect(LatLng(r.lat, r.lon))),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Text(
              _origin == null
                  ? 'Dodirni mapu ili pretraži adresu da postaviš polaznu tačku.'
                  : _destination == null
                      ? 'Dodirni mapu ili pretraži adresu da postaviš odredište.'
                      : 'Polazna i odredišna tačka su postavljene.',
              textAlign: TextAlign.center,
            ),
          ),
          Expanded(
            flex: 3,
            child: GoogleMap(
              initialCameraPosition: const CameraPosition(target: _serbiaCenter, zoom: 7),
              onMapCreated: (c) => _mapController = c,
              onTap: _handleTap,
              polylines: {
                ..._alternatePolylines,
                if (_routePoints.isNotEmpty)
                  Polyline(
                    polylineId: const PolylineId('route'),
                    points: _routePoints,
                    width: 4,
                    color: NocturneColors.accent,
                    zIndex: 1,
                  ),
              },
              markers: {
                if (_origin != null)
                  Marker(
                    markerId: const MarkerId('origin'),
                    position: _origin!,
                    icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueGreen),
                  ),
                if (_destination != null)
                  Marker(
                    markerId: const MarkerId('destination'),
                    position: _destination!,
                    icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueRed),
                  ),
              },
            ),
          ),
          // Own Scrollable (not the map) so a focused cargo field scrolls into
          // view above the keyboard instead of staying hidden under it.
          Expanded(
            flex: 2,
            child: SingleChildScrollView(
              child: Column(
                children: [
                  if (_error != null)
                    Padding(
                      padding: const EdgeInsets.all(8),
                      child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                    ),
                  if (_routeResult != null)
                    Card(
                      margin: const EdgeInsets.symmetric(horizontal: 12),
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child:
                            Text('${_routeResult!.distanceKm.toStringAsFixed(1)} km · ${_routeResult!.durationMin.toStringAsFixed(0)} min'),
                      ),
                    ),
                  Theme(
                    data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
                    child: ExpansionTile(
                      title: const Text('Podaci o tovaru (opciono)'),
                      leading: const Icon(Icons.inventory_2_outlined),
                      childrenPadding: const EdgeInsets.symmetric(horizontal: 16),
                      children: [
                        TextField(controller: _cargoDescriptionCtrl, decoration: const InputDecoration(labelText: 'Opis tovara')),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _cargoWeightCtrl,
                          keyboardType: const TextInputType.numberWithOptions(decimal: true),
                          decoration: const InputDecoration(labelText: 'Težina tovara (kg)'),
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _cargoTempRangeCtrl,
                          decoration: const InputDecoration(labelText: 'Temperaturni opseg', hintText: 'npr. -18°C do -15°C'),
                        ),
                        const SizedBox(height: 8),
                        TextField(controller: _pickupLocationCtrl, decoration: const InputDecoration(labelText: 'Mesto preuzimanja')),
                        const SizedBox(height: 8),
                        TextField(controller: _dropoffLocationCtrl, decoration: const InputDecoration(labelText: 'Mesto isporuke')),
                        const SizedBox(height: 12),
                      ],
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.all(12),
                    child: Row(
                      children: [
                        Expanded(
                          child: OutlinedButton(
                            onPressed: canAssign ? _previewRoute : null,
                            child: const Text('Pregled rute'),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: FilledButton(
                            onPressed: canAssign ? _assignTrip : null,
                            child: _loading
                                ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                                : const Text('Ponudi turu'),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
