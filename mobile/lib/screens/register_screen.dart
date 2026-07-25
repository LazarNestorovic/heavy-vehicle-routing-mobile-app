import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../services/auth_storage.dart';
import '../theme/nocturne_theme.dart';
import 'entry_router.dart';

class RegisterScreen extends StatefulWidget {
  final ApiClient api;
  const RegisterScreen({super.key, required this.api});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _formKey = GlobalKey<FormState>();
  final _usernameCtrl = TextEditingController();
  final _passwordCtrl = TextEditingController();
  final _authStorage = AuthStorage();

  String _role = 'driver';
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    _usernameCtrl.dispose();
    _passwordCtrl.dispose();
    super.dispose();
  }

  String? _validateUsername(String? v) {
    if (v == null || v.length < 3) return 'Minimum 3 karaktera';
    return null;
  }

  String? _validatePassword(String? v) {
    if (v == null || v.length < 6) return 'Minimum 6 karaktera';
    return null;
  }

  Future<void> _register() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final username = _usernameCtrl.text.trim();
      final result = await widget.api.register(username, _passwordCtrl.text, role: _role);
      widget.api.token = result.token;
      widget.api.driverId = result.driverId;
      widget.api.username = username;
      widget.api.role = result.role;
      widget.api.dispatcherId = result.dispatcherId;
      await _authStorage.save(result.token, result.driverId, username, result.role, result.dispatcherId);

      if (!mounted) return;
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => homeScreenFor(widget.api)),
      );
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (e) {
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Registracija')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Form(
            key: _formKey,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(value: 'driver', label: Text('Vozač'), icon: Icon(Icons.local_shipping)),
                    ButtonSegment(value: 'dispatcher', label: Text('Dispečer'), icon: Icon(Icons.business)),
                  ],
                  selected: {_role},
                  onSelectionChanged: (selected) => setState(() => _role = selected.first),
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _usernameCtrl,
                  decoration: const InputDecoration(labelText: 'Korisničko ime', border: OutlineInputBorder()),
                  validator: _validateUsername,
                ),
                const SizedBox(height: 12),
                TextFormField(
                  controller: _passwordCtrl,
                  decoration: const InputDecoration(labelText: 'Lozinka', border: OutlineInputBorder()),
                  obscureText: true,
                  validator: _validatePassword,
                ),
                const SizedBox(height: 20),
                if (_error != null)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                  ),
                FilledButton(
                  onPressed: _loading ? null : _register,
                  child: _loading
                      ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                      : const Text('Registruj se'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
