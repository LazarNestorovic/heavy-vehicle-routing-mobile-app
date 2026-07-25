import 'package:flutter/material.dart';

import '../models/vehicle_profile.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Vehicle creation form. Since a driver can own multiple vehicles now (see
/// documentations/features/2026-07-21-driver-preference-scoring.md, Faza 2),
/// this is no longer the app's first screen - VehicleListScreen is the hub,
/// this is reached via its "add vehicle" action and pops back with `true` on
/// success so the list knows to refresh.
class VehicleProfileScreen extends StatefulWidget {
  final ApiClient api;
  const VehicleProfileScreen({super.key, required this.api});

  @override
  State<VehicleProfileScreen> createState() => _VehicleProfileScreenState();
}

class _VehicleProfileScreenState extends State<VehicleProfileScreen> {
  final _formKey = GlobalKey<FormState>();

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

  Future<void> _submit() async {
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
      await widget.api.createVehicle(profile);

      if (!mounted) return;
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
    return Scaffold(
      appBar: AppBar(title: const Text('Novo vozilo')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
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
                  child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                ),
              FilledButton(
                onPressed: _loading ? null : _submit,
                child: _loading
                    ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text('Sačuvaj vozilo'),
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
