import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../services/google_auth.dart';
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
  final _emailCtrl = TextEditingController();
  final _googleAuth = GoogleAuthService();

  String _role = 'driver';
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    _usernameCtrl.dispose();
    _passwordCtrl.dispose();
    _emailCtrl.dispose();
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

  String? _validateEmail(String? v) {
    if (v == null || v.isEmpty) return 'Obavezno polje';
    if (!v.contains('@')) return 'Nevažeća email adresa';
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
      final email = _emailCtrl.text.trim();
      final result = await widget.api.register(username, _passwordCtrl.text, role: _role, email: email);
      await applySession(widget.api, result);

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

  Future<void> _continueWithGoogle() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final idToken = await _googleAuth.signIn();
      if (idToken == null) return; // user canceled - not an error
      final result = await widget.api.signInWithGoogle(idToken, role: _role);
      await applySession(widget.api, result);

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
                const SizedBox(height: 12),
                TextFormField(
                  controller: _emailCtrl,
                  decoration: const InputDecoration(
                    labelText: 'Email',
                    hintText: 'Za potvrdu naloga - poslaćemo link',
                    border: OutlineInputBorder(),
                  ),
                  keyboardType: TextInputType.emailAddress,
                  validator: _validateEmail,
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
                const SizedBox(height: 12),
                Row(
                  children: [
                    const Expanded(child: Divider()),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 8),
                      child: Text('ili', style: TextStyle(color: NocturneColors.text.withValues(alpha: 0.6))),
                    ),
                    const Expanded(child: Divider()),
                  ],
                ),
                const SizedBox(height: 12),
                OutlinedButton.icon(
                  onPressed: _loading ? null : _continueWithGoogle,
                  icon: const Icon(Icons.login),
                  label: const Text('Nastavi sa Google nalogom'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
