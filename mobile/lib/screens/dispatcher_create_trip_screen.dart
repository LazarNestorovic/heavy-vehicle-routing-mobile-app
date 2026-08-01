import 'dart:async';

import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../models/driver.dart';
import '../models/geocode_result.dart';
import '../models/route_result.dart';
import '../models/trip.dart';
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
///
/// Origin/destination picking mirrors RouteRequestScreen's redesign (separate
/// address fields, map tap, reverse geocoding, map-overlaying cargo panel)
/// with ONE deliberate difference: origin has NO "trenutna pozicija" GPS mode
/// and no special bottom-sheet picker - it's a plain address field exactly
/// like destination, since the dispatcher isn't the one driving (their
/// phone's GPS has nothing to do with where the driver/truck actually is,
/// and there's no reason to treat picking it any differently than the
/// destination).
///
/// Doubles as the edit screen for an already-offered/accepted trip when
/// [editing] is passed - same form, prefilled from the existing trip, PUT
/// instead of POST on submit (see ApiClient.updateTrip). If the trip was
/// "accepted", the backend reverts it to "offered" so the driver reviews and
/// re-accepts (see BELESKE.txt 2026-08-01 entry for the reasoning).
class DispatcherCreateTripScreen extends StatefulWidget {
  final ApiClient api;
  final Driver driver;
  final Trip? editing;
  const DispatcherCreateTripScreen({super.key, required this.api, required this.driver, this.editing});

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

  // Drives the bottom panel's height (see _bottomPanel) - collapsed by
  // default so the map gets most of the screen, grows to overlay the map
  // while the dispatcher is filling in cargo details.
  bool _cargoExpanded = false;

  final _originCtrl = TextEditingController();
  final _destinationCtrl = TextEditingController();
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

    final editing = widget.editing;
    if (editing != null) {
      _origin = LatLng(editing.originLat, editing.originLon);
      _destination = LatLng(editing.destinationLat, editing.destinationLon);
      unawaited(_reverseGeocodeOrigin(_origin!));
      unawaited(_reverseGeocodeDestination(_destination!));
      _cargoDescriptionCtrl.text = editing.cargoDescription ?? '';
      _cargoWeightCtrl.text = editing.cargoWeightKg?.toString() ?? '';
      _cargoTempRangeCtrl.text = editing.cargoTempRange ?? '';
      _pickupLocationCtrl.text = editing.pickupLocation ?? '';
      _dropoffLocationCtrl.text = editing.dropoffLocation ?? '';
      // The vehicle list loads async - once it's in, match the trip's
      // current vehicle by id so the dropdown starts pre-selected.
      _vehiclesFuture.then((vehicles) {
        if (!mounted) return;
        final match = vehicles.where((v) => v.id == editing.vehicleId);
        if (match.isNotEmpty) setState(() => _selectedVehicle = match.first);
      });
    }
  }

  @override
  void dispose() {
    _originCtrl.dispose();
    _destinationCtrl.dispose();
    _cargoDescriptionCtrl.dispose();
    _cargoWeightCtrl.dispose();
    _cargoTempRangeCtrl.dispose();
    _pickupLocationCtrl.dispose();
    _dropoffLocationCtrl.dispose();
    super.dispose();
  }

  // A map tap sets the origin first, then the destination on every tap after
  // that - same simple fallback as before origin/destination had their own
  // address fields. Either way, the tapped point only has raw coordinates, so
  // it's reverse-geocoded in the background to fill in a readable address
  // once one's found (best-effort).
  void _handleTap(LatLng point) {
    setState(() {
      _routeResult = null;
      _routePoints = [];
      _error = null;
      if (_origin == null) {
        _origin = point;
        _originCtrl.text = '';
        unawaited(_reverseGeocodeOrigin(point));
      } else {
        _destination = point;
        _destinationCtrl.text = '';
        unawaited(_reverseGeocodeDestination(point));
      }
    });
  }

  // Best-effort: if reverse geocoding fails (offline, no address found for
  // this point, etc.) the tapped point is still fully usable, it just leaves
  // the corresponding field empty instead of showing a readable address.
  Future<void> _reverseGeocodeOrigin(LatLng point) async {
    try {
      final result = await widget.api.reverseGeocode(point.latitude, point.longitude);
      if (!mounted || _origin != point) return; // origin changed again meanwhile - discard
      _originCtrl.text = result.displayName;
    } catch (_) {
      // Leave the field empty - origin point itself is still set and usable.
    }
  }

  Future<void> _reverseGeocodeDestination(LatLng point) async {
    try {
      final result = await widget.api.reverseGeocode(point.latitude, point.longitude);
      if (!mounted || _destination != point) return; // destination changed again meanwhile - discard
      _destinationCtrl.text = result.displayName;
    } catch (_) {
      // Leave the field empty - destination point itself is still set and usable.
    }
  }

  // Origin address search - always targets the origin, and recenters the map
  // on the picked point since it may be off-screen. Mirrors
  // _handleDestinationSelect exactly - origin and destination are picked the
  // same way (see class doc comment).
  void _handleOriginSelect(GeocodeResult result) {
    final point = LatLng(result.lat, result.lon);
    setState(() {
      _routeResult = null;
      _routePoints = [];
      _error = null;
      _origin = point;
    });
    _mapController?.animateCamera(CameraUpdate.newLatLng(point));
  }

  // Destination address search (see widgets/address_search_field.dart) -
  // always targets the destination, and recenters the map on the picked
  // point since it may be off-screen.
  void _handleDestinationSelect(GeocodeResult result) {
    final point = LatLng(result.lat, result.lon);
    setState(() {
      _routeResult = null;
      _routePoints = [];
      _error = null;
      _destination = point;
    });
    _mapController?.animateCamera(CameraUpdate.newLatLng(point));
  }

  void _reset() {
    setState(() {
      _origin = null;
      _originCtrl.text = '';
      _destination = null;
      _destinationCtrl.text = '';
      _routeResult = null;
      _routePoints = [];
      _error = null;
    });
  }

  Future<void> _previewRoute() async {
    if (_origin == null || _destination == null || _selectedVehicle == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result =
          await widget.api.previewRoute(origin: _origin!, destination: _destination!, vehicle: _selectedVehicle!);
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
    final editing = widget.editing;
    try {
      String message;
      if (editing != null) {
        final updated = await widget.api.updateTrip(
          tripId: editing.id,
          vehicleId: _selectedVehicle!.id!,
          origin: _origin!,
          destination: _destination!,
          cargoDescription: _cargoDescriptionCtrl.text.trim().isEmpty ? null : _cargoDescriptionCtrl.text.trim(),
          cargoWeightKg: double.tryParse(_cargoWeightCtrl.text.trim()),
          cargoTempRange: _cargoTempRangeCtrl.text.trim().isEmpty ? null : _cargoTempRangeCtrl.text.trim(),
          pickupLocation: _pickupLocationCtrl.text.trim().isEmpty ? null : _pickupLocationCtrl.text.trim(),
          dropoffLocation: _dropoffLocationCtrl.text.trim().isEmpty ? null : _dropoffLocationCtrl.text.trim(),
        );
        message = updated.status == 'offered' && editing.status == 'accepted'
            ? 'Tura izmenjena — vozač ${widget.driver.username} treba ponovo da je potvrdi.'
            : 'Tura izmenjena.';
      } else {
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
        message = 'Tura ponuđena vozaču ${widget.driver.username}.';
      }
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
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
      appBar: AppBar(
        title: Text(widget.editing != null ? 'Izmena ture — ${widget.driver.username}' : 'Nova tura — ${widget.driver.username}'),
        actions: [
          IconButton(onPressed: _reset, icon: const Icon(Icons.refresh), tooltip: 'Resetuj tačke'),
        ],
      ),
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
                  // Without this, the dropdown sizes itself to its widest
                  // item's intrinsic width instead of the available space -
                  // a long item label ("Flota · 4.0m/2.55m/16.5m · 40000kg")
                  // then overflows past the screen edge instead of wrapping
                  // to it (reported as "Right overflowed by 6.8 pixels").
                  isExpanded: true,
                  decoration: const InputDecoration(labelText: 'Vozilo', border: OutlineInputBorder()),
                  items: vehicles
                      .map((v) => DropdownMenuItem(
                            value: v,
                            child: Text(
                              '${v.isFleet ? "Flota" : widget.driver.username} · '
                              '${v.heightM}m/${v.widthM}m/${v.lengthM}m · ${v.weightKg.toStringAsFixed(0)}kg',
                              overflow: TextOverflow.ellipsis,
                            ),
                          ))
                      .toList(),
                  onChanged: (v) => setState(() => _selectedVehicle = v),
                );
              },
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
            child: Column(
              children: [
                AddressSearchField(
                  api: widget.api,
                  controller: _originCtrl,
                  hintText: 'Polazna tačka (adresa)',
                  onSelected: _handleOriginSelect,
                ),
                const SizedBox(height: 8),
                AddressSearchField(
                  api: widget.api,
                  controller: _destinationCtrl,
                  hintText: 'Odredište (adresa)',
                  onSelected: _handleDestinationSelect,
                ),
              ],
            ),
          ),
          // The map fills essentially the whole remaining screen; the panel
          // below floats OVER it (Stack, not a flex sibling) so the map isn't
          // permanently squeezed to make room for cargo details that are only
          // relevant while expanded - see _bottomPanel.
          Expanded(
            child: Stack(
              children: [
                Positioned.fill(
                  child: GoogleMap(
                    initialCameraPosition: const CameraPosition(target: _serbiaCenter, zoom: 7),
                    onMapCreated: (c) {
                      _mapController = c;
                      if (_origin != null) {
                        c.animateCamera(CameraUpdate.newLatLngZoom(_origin!, 7));
                      }
                    },
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
                Positioned(left: 0, right: 0, bottom: 0, child: _bottomPanel(context, canAssign)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // Compact by default (error/summary/collapsed cargo header) so the map
  // stays visible above it; grows to overlay much more of the map while the
  // cargo section is expanded, so there's room to comfortably fill in cargo
  // details instead of fighting a cramped, fixed-size strip. Buttons stay
  // pinned to the bottom of the panel either way (see the inner
  // Expanded+SingleChildScrollView, sticky-footer pattern).
  Widget _bottomPanel(BuildContext context, bool canAssign) {
    final expandedHeight = MediaQuery.sizeOf(context).height * 0.6;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeInOut,
      constraints: BoxConstraints(maxHeight: _cargoExpanded ? expandedHeight : 120),
      decoration: BoxDecoration(
        color: Theme.of(context).scaffoldBackgroundColor,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
        boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 0.25), blurRadius: 8, offset: const Offset(0, -2))],
      ),
      child: Column(
        children: [
          Expanded(
            child: SingleChildScrollView(
              child: Column(
                children: [
                  if (_error != null)
                    Padding(
                      padding: const EdgeInsets.fromLTRB(12, 8, 12, 0),
                      child: Text(_error!, style: const TextStyle(color: NocturneColors.error, fontSize: 13)),
                    ),
                  if (_routeResult != null)
                    Card(
                      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                        child: Text(
                          '${_routeResult!.distanceKm.toStringAsFixed(1)} km · ${_routeResult!.durationMin.toStringAsFixed(0)} min',
                          style: Theme.of(context).textTheme.titleSmall,
                        ),
                      ),
                    ),
                  Theme(
                    data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
                    child: ExpansionTile(
                      title: const Text('Podaci o tovaru (opciono)'),
                      leading: const Icon(Icons.inventory_2_outlined),
                      childrenPadding: const EdgeInsets.symmetric(horizontal: 16),
                      onExpansionChanged: (expanded) => setState(() => _cargoExpanded = expanded),
                      children: [
                        TextField(
                            controller: _cargoDescriptionCtrl,
                            decoration: const InputDecoration(labelText: 'Opis tovara')),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _cargoWeightCtrl,
                          keyboardType: const TextInputType.numberWithOptions(decimal: true),
                          decoration: const InputDecoration(labelText: 'Težina tovara (kg)'),
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _cargoTempRangeCtrl,
                          decoration:
                              const InputDecoration(labelText: 'Temperaturni opseg', hintText: 'npr. -18°C do -15°C'),
                        ),
                        const SizedBox(height: 8),
                        TextField(
                            controller: _pickupLocationCtrl,
                            decoration: const InputDecoration(labelText: 'Mesto preuzimanja')),
                        const SizedBox(height: 8),
                        TextField(
                            controller: _dropoffLocationCtrl,
                            decoration: const InputDecoration(labelText: 'Mesto isporuke')),
                        const SizedBox(height: 12),
                      ],
                    ),
                  ),
                ],
              ),
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
                        : Text(widget.editing != null ? 'Sačuvaj izmene' : 'Ponudi turu'),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
