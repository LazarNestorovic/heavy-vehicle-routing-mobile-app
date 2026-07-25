import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';

import '../models/driver_preferences.dart';
import '../models/favorite_stop.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Driver preference sliders (1-5, see documentations/features/2026-07-21-driver-preference-scoring.md)
/// plus preferred fuel brand and saved favorite stops (documentations/features/
/// 2026-07-21-preferred-fuel-stations.md). 3 is "neutral" on every slider.
class PreferencesScreen extends StatefulWidget {
  final ApiClient api;
  const PreferencesScreen({super.key, required this.api});

  @override
  State<PreferencesScreen> createState() => _PreferencesScreenState();
}

class _PreferencesScreenState extends State<PreferencesScreen> {
  DriverPreferences _prefs = DriverPreferences.neutral;
  final _brandCtrl = TextEditingController();
  List<FavoriteStop> _favorites = [];

  bool _loading = true;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _brandCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final prefs = await widget.api.getPreferences();
      final favorites = await widget.api.listFavoriteStops();
      setState(() {
        _prefs = prefs;
        _brandCtrl.text = prefs.preferredFuelBrand ?? '';
        _favorites = favorites;
      });
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final updated = await widget.api.updatePreferences(_prefs.copyWith(preferredFuelBrand: _brandCtrl.text.trim()));
      setState(() => _prefs = updated);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Preference sačuvane')));
      }
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _addFavorite() async {
    final picked = await Navigator.of(context).push<_PickedStop>(
      MaterialPageRoute(builder: (_) => const _FavoriteStopPickerScreen()),
    );
    if (picked == null) return;

    try {
      final saved = await widget.api.createFavoriteStop(lat: picked.point.latitude, lon: picked.point.longitude, name: picked.name);
      setState(() => _favorites = [saved, ..._favorites]);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    }
  }

  Future<void> _deleteFavorite(FavoriteStop stop) async {
    try {
      await widget.api.deleteFavoriteStop(stop.id);
      setState(() => _favorites.removeWhere((f) => f.id == stop.id));
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Preference')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                if (_error != null)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                  ),
                _prioritySlider(
                  'Ušteda goriva',
                  _prefs.fuelPriority,
                  (v) => setState(() => _prefs = _prefs.copyWith(fuelPriority: v)),
                ),
                _prioritySlider(
                  'Osetljivost tovara',
                  _prefs.cargoPriority,
                  (v) => setState(() => _prefs = _prefs.copyWith(cargoPriority: v)),
                ),
                _prioritySlider(
                  'Udeo auto-puta',
                  _prefs.highwayPriority,
                  (v) => setState(() => _prefs = _prefs.copyWith(highwayPriority: v)),
                ),
                _prioritySlider(
                  'Brzina / vreme putovanja',
                  _prefs.timePriority,
                  (v) => setState(() => _prefs = _prefs.copyWith(timePriority: v)),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _brandCtrl,
                  decoration: const InputDecoration(
                    labelText: 'Preferirani brend pumpe (opciono)',
                    hintText: 'npr. НИС Петрол',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: _saving ? null : _save,
                  child: _saving
                      ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                      : const Text('Sačuvaj preference'),
                ),
                const Divider(height: 40),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text('Omiljene lokacije', style: Theme.of(context).textTheme.titleMedium),
                    IconButton(onPressed: _addFavorite, icon: const Icon(Icons.add_location_alt)),
                  ],
                ),
                if (_favorites.isEmpty) const Text('Nema sačuvanih lokacija.'),
                for (final stop in _favorites)
                  ListTile(
                    leading: const Icon(Icons.star, color: Colors.amber),
                    title: Text(stop.name.isEmpty ? 'Lokacija #${stop.id}' : stop.name),
                    subtitle: Text('${stop.lat.toStringAsFixed(5)}, ${stop.lon.toStringAsFixed(5)}'),
                    trailing: IconButton(
                      icon: const Icon(Icons.delete_outline),
                      onPressed: () => _deleteFavorite(stop),
                    ),
                  ),
              ],
            ),
    );
  }

  Widget _prioritySlider(String label, int value, ValueChanged<int> onChanged) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('$label: $value/5'),
          Slider(
            value: value.toDouble(),
            min: 1,
            max: 5,
            divisions: 4,
            label: '$value',
            onChanged: (v) => onChanged(v.round()),
          ),
        ],
      ),
    );
  }
}

class _PickedStop {
  final LatLng point;
  final String name;
  _PickedStop(this.point, this.name);
}

/// Minimal tap-to-place picker for a new favorite stop.
class _FavoriteStopPickerScreen extends StatefulWidget {
  const _FavoriteStopPickerScreen();

  @override
  State<_FavoriteStopPickerScreen> createState() => _FavoriteStopPickerScreenState();
}

class _FavoriteStopPickerScreenState extends State<_FavoriteStopPickerScreen> {
  final _nameCtrl = TextEditingController();
  LatLng? _point;

  @override
  void dispose() {
    _nameCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Dodaj omiljenu lokaciju')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: Text(
              _point == null ? 'Dodirni mapu da izabereš lokaciju.' : 'Lokacija izabrana.',
              textAlign: TextAlign.center,
            ),
          ),
          Expanded(
            child: FlutterMap(
              options: MapOptions(
                initialCenter: const LatLng(44.5, 20.5),
                initialZoom: 7,
                onTap: (tapPos, point) => setState(() => _point = point),
              ),
              children: [
                TileLayer(
                  urlTemplate: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
                  userAgentPackageName: 'com.example.hvr_mobile',
                ),
                MarkerLayer(markers: [
                  if (_point != null)
                    Marker(point: _point!, width: 40, height: 40, child: const Icon(Icons.star, color: Colors.amber)),
                ]),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextField(
                  controller: _nameCtrl,
                  decoration: const InputDecoration(labelText: 'Naziv (opciono)', border: OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                FilledButton(
                  onPressed: _point == null
                      ? null
                      : () => Navigator.of(context).pop(_PickedStop(_point!, _nameCtrl.text.trim())),
                  child: const Text('Sačuvaj lokaciju'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
