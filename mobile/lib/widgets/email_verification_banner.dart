import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';

/// Shown on every role's home screen when the account has an email on file
/// that hasn't been verified yet (see documentations/features/ entry).
/// Renders nothing otherwise. The only way the app currently learns
/// verification happened is a fresh login, since the emailed link opens in a
/// browser, not back into the app (no deep-linking - a deliberate scope cut,
/// noted directly in the success message below).
class EmailVerificationBanner extends StatefulWidget {
  final ApiClient api;
  const EmailVerificationBanner({super.key, required this.api});

  @override
  State<EmailVerificationBanner> createState() => _EmailVerificationBannerState();
}

class _EmailVerificationBannerState extends State<EmailVerificationBanner> {
  bool _sending = false;
  String? _status;

  Future<void> _resend() async {
    setState(() {
      _sending = true;
      _status = null;
    });
    try {
      await widget.api.resendVerificationEmail();
      if (!mounted) return;
      setState(() => _status = 'Poslato! Klikni link u email-u, pa se ponovo uloguj da se status osveži.');
    } catch (e) {
      if (!mounted) return;
      setState(() => _status = 'Greška: $e');
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.api.email == null || widget.api.emailVerified) return const SizedBox.shrink();

    return Card(
      margin: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      color: NocturneColors.accent800,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.mark_email_unread_outlined, color: NocturneColors.accent300),
                const SizedBox(width: 8),
                const Expanded(child: Text('Email adresa nije potvrđena.')),
                TextButton(
                  onPressed: _sending ? null : _resend,
                  child: _sending
                      ? const SizedBox(height: 16, width: 16, child: CircularProgressIndicator(strokeWidth: 2))
                      : const Text('Pošalji ponovo'),
                ),
              ],
            ),
            if (_status != null) Padding(padding: const EdgeInsets.only(top: 4), child: Text(_status!)),
          ],
        ),
      ),
    );
  }
}
