import 'package:flutter/material.dart';

import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../services/vehicle_storage.dart';
import 'route_request_screen.dart';

class VehicleProfileScreen extends StatefulWidget {
  const VehicleProfileScreen({super.key});

  @override
  State<VehicleProfileScreen> createState() => _VehicleProfileScreenState();
}

class _VehicleProfileScreenState extends State<VehicleProfileScreen> {
  final _formKey = GlobalKey<FormState>();
  final _storage = VehicleStorage();
  final _api = ApiClient();

  // Pre-filled with a typical EU semi-truck profile so the form is usable
  // immediately during a demo, not just after manual entry.
  final _heightCtrl = TextEditingController(text: '4.0');
  final _widthCtrl = TextEditingController(text: '2.55');
  final _lengthCtrl = TextEditingController(text: '16.5');
  final _weightCtrl = TextEditingController(text: '40000');
  final _axleLoadCtrl = TextEditingController(text: '11500');
  bool _hazmat = false;

  bool _loading = false;
  String? _error;
  VehicleProfile? _saved;

  @override
  void initState() {
    super.initState();
    _loadSaved();
  }

  Future<void> _loadSaved() async {
    final saved = await _storage.load();
    if (saved != null && mounted) {
      setState(() {
        _saved = saved;
        _heightCtrl.text = saved.heightM.toString();
        _widthCtrl.text = saved.widthM.toString();
        _lengthCtrl.text = saved.lengthM.toString();
        _weightCtrl.text = saved.weightKg.toString();
        _axleLoadCtrl.text = saved.axleLoadKg.toString();
        _hazmat = saved.hazmat;
      });
    }
  }

  @override
  void dispose() {
    _heightCtrl.dispose();
    _widthCtrl.dispose();
    _lengthCtrl.dispose();
    _weightCtrl.dispose();
    _axleLoadCtrl.dispose();
    super.dispose();
  }

  double _num(TextEditingController c) => double.parse(c.text.replaceAll(',', '.'));

  String? _validatePositive(String? value) {
    if (value == null || value.isEmpty) return 'Obavezno polje';
    final n = double.tryParse(value.replaceAll(',', '.'));
    if (n == null || n <= 0) return 'Unesite pozitivan broj';
    return null;
  }

  Future<void> _continue() async {
    if (!_formKey.currentState!.validate()) return;

    final weight = _num(_weightCtrl);
    final axleLoad = _num(_axleLoadCtrl);
    if (axleLoad > weight) {
      setState(() => _error = 'Osovinsko opterećenje ne može biti veće od ukupne težine vozila');
      return;
    }

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final profile = VehicleProfile(
        heightM: _num(_heightCtrl),
        widthM: _num(_widthCtrl),
        lengthM: _num(_lengthCtrl),
        weightKg: weight,
        axleLoadKg: axleLoad,
        hazmat: _hazmat,
      );
      final saved = await _api.createVehicle(profile);
      await _storage.save(saved);

      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(builder: (_) => RouteRequestScreen(vehicle: saved)),
      );
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
    return Scaffold(
      appBar: AppBar(title: const Text('Profil vozila')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (_saved != null)
                const Padding(
                  padding: EdgeInsets.only(bottom: 12),
                  child: Text(
                    'Prethodno sačuvan profil je učitan. Izmenite polja po potrebi ili nastavite.',
                    style: TextStyle(fontStyle: FontStyle.italic),
                  ),
                ),
              _numberField('Visina (m)', _heightCtrl),
              _numberField('Širina (m)', _widthCtrl),
              _numberField('Dužina (m)', _lengthCtrl),
              _numberField('Težina (kg)', _weightCtrl),
              _numberField('Osovinsko opterećenje (kg)', _axleLoadCtrl),
              SwitchListTile(
                title: const Text('Prevoz opasnog tereta (hazmat)'),
                value: _hazmat,
                onChanged: (v) => setState(() => _hazmat = v),
              ),
              const SizedBox(height: 16),
              if (_error != null)
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: Text(_error!, style: const TextStyle(color: Colors.red)),
                ),
              FilledButton(
                onPressed: _loading ? null : _continue,
                child: _loading
                    ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text('Sačuvaj i nastavi'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _numberField(String label, TextEditingController controller) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextFormField(
        controller: controller,
        decoration: InputDecoration(labelText: label, border: const OutlineInputBorder()),
        keyboardType: const TextInputType.numberWithOptions(decimal: true),
        validator: _validatePositive,
      ),
    );
  }
}
