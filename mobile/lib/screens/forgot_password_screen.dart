import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Just an email field - the actual password reset happens on a web page
/// opened from the emailed link (backend/internal/httpapi/auth.go
/// handleShowResetPasswordForm), same "browser, not deep-link" pattern as
/// email verification. See documentations/features/ entry.
class ForgotPasswordScreen extends StatefulWidget {
  final ApiClient api;
  // Pre-fills the email field when opened from a screen that already knows
  // the caller's address (see ProfileScreen's "Promeni lozinku" button) -
  // still editable, since the account may not be the one whose password
  // needs resetting.
  final String? initialEmail;
  const ForgotPasswordScreen({super.key, required this.api, this.initialEmail});

  @override
  State<ForgotPasswordScreen> createState() => _ForgotPasswordScreenState();
}

class _ForgotPasswordScreenState extends State<ForgotPasswordScreen> {
  final _formKey = GlobalKey<FormState>();
  late final _emailCtrl = TextEditingController(text: widget.initialEmail);
  bool _loading = false;
  bool _sent = false;
  String? _error;

  @override
  void dispose() {
    _emailCtrl.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      await widget.api.forgotPassword(_emailCtrl.text.trim());
      if (!mounted) return;
      setState(() => _sent = true);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Neočekivana greška: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Zaboravljena lozinka')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: _sent
              ? const Text(
                  'Ako nalog sa ovim mejlom postoji, poslat je link za resetovanje lozinke. Proveri inbox i klikni link.',
                  textAlign: TextAlign.center,
                )
              : Form(
                  key: _formKey,
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const Text('Unesi email sa kojim si se registrovao/la - poslaćemo link za resetovanje lozinke.'),
                      const SizedBox(height: 16),
                      TextFormField(
                        controller: _emailCtrl,
                        decoration: const InputDecoration(labelText: 'Email', border: OutlineInputBorder()),
                        keyboardType: TextInputType.emailAddress,
                        validator: (v) => (v == null || !v.contains('@')) ? 'Nevažeća email adresa' : null,
                      ),
                      const SizedBox(height: 20),
                      if (_error != null)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 12),
                          child: Text(_error!, style: const TextStyle(color: NocturneColors.error)),
                        ),
                      FilledButton(
                        onPressed: _loading ? null : _submit,
                        child: _loading
                            ? const SizedBox(height: 20, width: 20, child: CircularProgressIndicator(strokeWidth: 2))
                            : const Text('Pošalji link'),
                      ),
                    ],
                  ),
                ),
        ),
      ),
    );
  }
}
