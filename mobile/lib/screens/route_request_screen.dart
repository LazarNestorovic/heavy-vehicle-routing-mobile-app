import 'dart:async';

import 'package:flutter/material.dart';
import 'package:geolocator/geolocator.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../models/geocode_result.dart';
import '../models/route_result.dart';
import '../models/trip.dart';
import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/location_service.dart';
import '../services/polyline.dart';
import '../services/route_observer.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/address_search_field.dart';
import '../widgets/start_proximity_status.dart';
import 'active_trip_screen.dart';

/// Map + route request screen (SPECIFIKACIJA.md 3.9). Origin defaults to
/// "trenutna pozicija" (a live mode, not a frozen point - it tracks the
/// driver's GPS as it moves) until explicitly overridden by typing an
/// address or picking a point on the map (see _showOriginPicker); the
/// destination is set the same way it always was - type an address or tap
/// the map. Previewing works from anywhere; actually STARTING requires being
/// at the origin (see widgets/start_proximity_status.dart) - mirrors
/// turn-by-turn navigation apps, added after live GPS tracking made a
/// mismatch between the planned origin and the driver's real position
/// visibly confusing.
class RouteRequestScreen extends StatefulWidget {
  final ApiClient api;
  final VehicleProfile vehicle;
  const RouteRequestScreen({super.key, required this.api, required this.vehicle});

  @override
  State<RouteRequestScreen> createState() => _RouteRequestScreenState();
}

class _RouteRequestScreenState extends State<RouteRequestScreen> with RouteAware {
  final _locationService = LocationService();
  StreamSubscription<Position>? _positionSub;

  GoogleMapController? _mapController;
  LatLng? _myPosition;

  // _origin == null means "trenutna pozicija" mode - see _effectiveOrigin.
  // _originLabel is the address text to show for an address-search pick;
  // null with _origin set means it came from a map tap instead.
  LatLng? _origin;
  String? _originLabel;
  bool _pickingOriginOnMap = false;

  LatLng? _destination;
  RouteResult? _routeResult;
  List<LatLng> _routePoints = [];
  bool _canStart = false;

  // Drives the bottom panel's height (see _bottomPanel) - collapsed by
  // default so the map gets most of the screen, grows to overlay the map
  // while the driver is filling in cargo details.
  bool _cargoExpanded = false;

  final _destinationCtrl = TextEditingController();
  final _cargoDescriptionCtrl = TextEditingController();
  final _cargoWeightCtrl = TextEditingController();
  final _cargoTempRangeCtrl = TextEditingController();
  final _pickupLocationCtrl = TextEditingController();
  final _dropoffLocationCtrl = TextEditingController();

  bool _loading = false;
  String? _error;

  // Blocks planning a new route while another trip is already active - mirrors
  // the backend's own 409 in handleCreateTrip (see widgets/active_trip_banner.dart
  // for the other half: a way back to an active trip from the home screen).
  Trip? _activeTrip;

  // The actual point used for preview/start: the explicitly chosen origin if
  // there is one, otherwise the driver's live GPS position.
  LatLng? get _effectiveOrigin => _origin ?? _myPosition;

  String get _originFieldText {
    if (_origin == null) return 'Trenutna pozicija';
    return _originLabel ??
        'Izabrana tačka (${_origin!.latitude.toStringAsFixed(4)}, ${_origin!.longitude.toStringAsFixed(4)})';
  }

  @override
  void initState() {
    super.initState();
    unawaited(_initLocation());
    unawaited(_checkActiveTrip());
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final route = ModalRoute.of(context);
    if (route is PageRoute) routeObserver.subscribe(this, route);
  }

  // Fires when a route pushed on top of this screen (ActiveTripScreen, after
  // "Kreni na put") is popped and this screen becomes visible again - Flutter
  // does NOT call initState() again for it, so without this the screen kept
  // showing "no active trip" (letting the driver plan/start another one)
  // right after backing out of the one they just started, until the whole
  // app was restarted.
  @override
  void didPopNext() => _checkActiveTrip();

  Future<void> _checkActiveTrip() async {
    try {
      final trip = await widget.api.findActiveTrip();
      if (!mounted) return;
      setState(() => _activeTrip = trip);
    } catch (_) {
      // Best-effort - if the check itself fails, don't block planning on it;
      // the backend's own 409 is still the real enforcement either way.
    }
  }

  // Keeps a "you are here" marker updating continuously on the map, even
  // before any route is planned/started (per the driver's explicit request),
  // and feeds _effectiveOrigin while origin is in "trenutna pozicija" mode.
  Future<void> _initLocation() async {
    if (await _locationService.ensurePermission() != GpsStatus.granted) return;
    final position = await _locationService.getCurrentPosition();
    if (position != null && mounted) {
      final point = LatLng(position.latitude, position.longitude);
      setState(() => _myPosition = point);
      _mapController?.animateCamera(CameraUpdate.newLatLngZoom(point, 13));
    }
    _positionSub = _locationService.positionStream().listen((p) {
      if (!mounted) return;
      setState(() => _myPosition = LatLng(p.latitude, p.longitude));
    });
  }

  void _handleMapCreated(GoogleMapController controller) {
    _mapController = controller;
    if (_myPosition != null) {
      controller.animateCamera(CameraUpdate.newLatLngZoom(_myPosition!, 13));
    }
  }

  @override
  void dispose() {
    routeObserver.unsubscribe(this);
    _positionSub?.cancel();
    _destinationCtrl.dispose();
    _cargoDescriptionCtrl.dispose();
    _cargoWeightCtrl.dispose();
    _cargoTempRangeCtrl.dispose();
    _pickupLocationCtrl.dispose();
    _dropoffLocationCtrl.dispose();
    super.dispose();
  }

  static const _serbiaCenter = LatLng(44.5, 20.5);

  // Not-chosen alternatives from the last preview, drawn thin/grey underneath
  // the chosen route (see documentations/features/ entry) - purely visual,
  // no effect on which route gets started.
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

  // A map tap sets the destination, exactly as before - UNLESS the driver
  // just armed "izaberi tačku na mapi" from the origin picker (see
  // _showOriginPicker), in which case this one tap sets the origin instead
  // and disarms picking mode. Either way, the tapped point only has raw
  // coordinates, so it's reverse-geocoded in the background to fill in a
  // readable address once one's found (best-effort - see _reverseGeocode*).
  void _handleTap(LatLng point) {
    setState(() {
      _routeResult = null;
      _routePoints = [];
      _error = null;
      if (_pickingOriginOnMap) {
        _origin = point;
        _originLabel = null;
        _pickingOriginOnMap = false;
        _canStart = false;
        unawaited(_reverseGeocodeOrigin(point));
      } else {
        _destination = point;
        _destinationCtrl.text = '';
        unawaited(_reverseGeocodeDestination(point));
      }
    });
  }

  // Best-effort: if reverse geocoding fails (offline, no address found for
  // this point, etc.) the tapped point is still fully usable, it just keeps
  // showing coordinates/an empty field instead of a readable address.
  Future<void> _reverseGeocodeOrigin(LatLng point) async {
    try {
      final result = await widget.api.reverseGeocode(point.latitude, point.longitude);
      if (!mounted || _origin != point) return; // origin changed again meanwhile - discard
      setState(() => _originLabel = result.displayName);
    } catch (_) {
      // Leave the coordinate fallback from _originFieldText.
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

  // Destination address search (see widgets/address_search_field.dart) -
  // unchanged from before, always targets the destination, and recenters the
  // map on the picked point since it may be off-screen.
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

  // Origin address search, from the bottom sheet opened by _showOriginPicker -
  // sets an explicit origin (leaving "trenutna pozicija" mode) and closes the sheet.
  void _handleOriginAddressSelect(GeocodeResult result) {
    final point = LatLng(result.lat, result.lon);
    setState(() {
      _origin = point;
      _originLabel = result.displayName;
      _pickingOriginOnMap = false;
      _routeResult = null;
      _routePoints = [];
      _canStart = false;
      _error = null;
    });
    Navigator.of(context).pop();
    _mapController?.animateCamera(CameraUpdate.newLatLng(point));
  }

  // Opens the "Polazna tačka" picker: type an address, pick a point on the
  // map, or go back to "trenutna pozicija" (live GPS).
  Future<void> _showOriginPicker() async {
    await showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => SingleChildScrollView(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 16,
          bottom: MediaQuery.of(sheetContext).viewInsets.bottom + 16,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Polazna tačka', style: Theme.of(sheetContext).textTheme.titleMedium),
            const SizedBox(height: 12),
            AddressSearchField(api: widget.api, hintText: 'Pretraga adrese', onSelected: _handleOriginAddressSelect),
            const SizedBox(height: 12),
            ListTile(
              leading: const Icon(Icons.my_location),
              title: const Text('Trenutna pozicija'),
              onTap: () {
                setState(() {
                  _origin = null;
                  _originLabel = null;
                  _pickingOriginOnMap = false;
                  _routeResult = null;
                  _routePoints = [];
                  _canStart = false;
                  _error = null;
                });
                Navigator.of(sheetContext).pop();
              },
            ),
            ListTile(
              leading: const Icon(Icons.map_outlined),
              title: const Text('Izaberi tačku na mapi'),
              onTap: () {
                setState(() => _pickingOriginOnMap = true);
                Navigator.of(sheetContext).pop();
                ScaffoldMessenger.of(context)
                    .showSnackBar(const SnackBar(content: Text('Dodirni mapu da postaviš polaznu tačku.')));
              },
            ),
          ],
        ),
      ),
    );
  }

  void _reset() {
    setState(() {
      _origin = null;
      _originLabel = null;
      _pickingOriginOnMap = false;
      _destination = null;
      _destinationCtrl.text = '';
      _routeResult = null;
      _routePoints = [];
      _error = null;
      _canStart = false;
    });
  }

  Future<void> _previewRoute() async {
    final origin = _effectiveOrigin;
    if (origin == null || _destination == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.api.previewRoute(
        origin: origin,
        destination: _destination!,
        vehicle: widget.vehicle,
      );
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

  Future<void> _startTrip() async {
    final origin = _effectiveOrigin;
    if (origin == null || _destination == null || widget.vehicle.id == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final trip = await widget.api.createTrip(
        vehicleId: widget.vehicle.id!,
        origin: origin,
        destination: _destination!,
        cargoDescription: _cargoDescriptionCtrl.text.trim().isEmpty ? null : _cargoDescriptionCtrl.text.trim(),
        cargoWeightKg: double.tryParse(_cargoWeightCtrl.text.trim()),
        cargoTempRange: _cargoTempRangeCtrl.text.trim().isEmpty ? null : _cargoTempRangeCtrl.text.trim(),
        pickupLocation: _pickupLocationCtrl.text.trim().isEmpty ? null : _pickupLocationCtrl.text.trim(),
        dropoffLocation: _dropoffLocationCtrl.text.trim().isEmpty ? null : _dropoffLocationCtrl.text.trim(),
      );
      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => ActiveTripScreen(api: widget.api, trip: trip, vehicleId: widget.vehicle.id!)),
      );
    } on ApiException catch (e) {
      setState(() => _error = 'Greška: ${e.message}');
    } catch (e) {
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  // Shown when the driver already has another trip "created"/"in_progress" -
  // previewing a route is still harmless and left available, but starting
  // one is blocked (see the "Kreni na put" button's onPressed condition) -
  // backend independently rejects it too (see handleCreateTrip's 409).
  Widget _activeTripBlockBanner(Trip trip) {
    return Card(
      margin: const EdgeInsets.fromLTRB(8, 8, 8, 0),
      color: NocturneColors.accent800,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Row(
          children: [
            const Icon(Icons.local_shipping, color: NocturneColors.accent300),
            const SizedBox(width: 8),
            const Expanded(child: Text('Već imate aktivnu turu u toku - prvo je završite.')),
            FilledButton(
              onPressed: () => Navigator.of(context).push(
                MaterialPageRoute(
                    builder: (_) => ActiveTripScreen(api: widget.api, trip: trip, vehicleId: trip.vehicleId)),
              ),
              child: const Text('Idi na nju'),
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Nova ruta'),
        actions: [
          IconButton(onPressed: _reset, icon: const Icon(Icons.refresh), tooltip: 'Resetuj tačke'),
        ],
      ),
      body: Column(
        children: [
          if (_activeTrip != null) _activeTripBlockBanner(_activeTrip!),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 5),
            child: Column(
              children: [
                InkWell(
                  onTap: _showOriginPicker,
                  child: InputDecorator(
                    decoration: const InputDecoration(
                      labelText: 'Polazna tačka',
                      border: OutlineInputBorder(),
                      isDense: true,
                      prefixIcon: Icon(Icons.trip_origin),
                      suffixIcon: Icon(Icons.expand_more),
                    ),
                    child: Text(
                      _pickingOriginOnMap ? 'Dodirni mapu…' : _originFieldText,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
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
                    onMapCreated: _handleMapCreated,
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
                      if (_myPosition != null)
                        Marker(
                          markerId: const MarkerId('my_position'),
                          position: _myPosition!,
                          icon: BitmapDescriptor.defaultMarkerWithHue(BitmapDescriptor.hueAzure),
                          anchor: const Offset(0.5, 0.5),
                          flat: true,
                          zIndex: 2,
                          infoWindow: const InfoWindow(title: 'Vaša pozicija'),
                        ),
                      if (_effectiveOrigin != null)
                        Marker(
                          markerId: const MarkerId('origin'),
                          position: _effectiveOrigin!,
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
                Positioned(left: 0, right: 0, bottom: 0, child: _bottomPanel(context)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  // Compact by default (error/summary/proximity/collapsed cargo header) so
  // the map stays visible above it; grows to overlay much more of the map
  // while the cargo section is expanded, so there's room to comfortably fill
  // in cargo details instead of fighting a cramped, fixed-size strip. Buttons
  // stay pinned to the bottom of the panel either way (see the inner
  // Expanded+SingleChildScrollView, same sticky-footer pattern as before).
  Widget _bottomPanel(BuildContext context) {
    final expandedHeight = MediaQuery.sizeOf(context).height * 0.6;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeInOut,
      constraints: BoxConstraints(maxHeight: _cargoExpanded ? expandedHeight : 200),
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
                  if (_routeResult != null) _RouteSummaryCard(result: _routeResult!),
                  if (_effectiveOrigin != null)
                    StartProximityStatus(
                      key: ValueKey('${_effectiveOrigin!.latitude},${_effectiveOrigin!.longitude}'),
                      originLat: _effectiveOrigin!.latitude,
                      originLon: _effectiveOrigin!.longitude,
                      onCanStartChanged: (canStart) => setState(() => _canStart = canStart),
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
                          decoration: const InputDecoration(labelText: 'Opis tovara'),
                        ),
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
                          decoration: const InputDecoration(labelText: 'Mesto preuzimanja'),
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _dropoffLocationCtrl,
                          decoration: const InputDecoration(labelText: 'Mesto isporuke'),
                        ),
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
                    onPressed: (_effectiveOrigin != null && _destination != null && !_loading) ? _previewRoute : null,
                    child: const Text('Pregled rute'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: (_effectiveOrigin != null &&
                            _destination != null &&
                            !_loading &&
                            _canStart &&
                            _activeTrip == null)
                        ? _startTrip
                        : null,
                    child: _loading
                        ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Text('Kreni na put'),
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

class _RouteSummaryCard extends StatelessWidget {
  final RouteResult result;
  const _RouteSummaryCard({required this.result});

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('${result.distanceKm.toStringAsFixed(1)} km · ${result.durationMin.toStringAsFixed(0)} min',
                style: Theme.of(context).textTheme.titleSmall),
            Text('Risk score: ${result.riskScore.toStringAsFixed(1)}', style: const TextStyle(fontSize: 12)),
            if (result.explanation != null) ...[
              const SizedBox(height: 4),
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Icon(Icons.info_outline, size: 16, color: NocturneColors.accent300),
                  const SizedBox(width: 6),
                  Expanded(
                    child: Text(result.explanation!,
                        style: const TextStyle(fontStyle: FontStyle.italic, fontSize: 12),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis),
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}
