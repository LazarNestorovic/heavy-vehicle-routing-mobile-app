import 'package:flutter/material.dart';

import '../models/account_status.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import 'entry_router.dart';
import 'forgot_password_screen.dart';
import 'login_screen.dart';

/// Mock-up's "Profile" panel: account identity + sign out. No shift-tracking
/// backend exists (see truck_status_screen.dart for driving hours, which is
/// the closest real equivalent) - this stays intentionally simple.
class ProfileScreen extends StatefulWidget {
  final ApiClient api;
  const ProfileScreen({super.key, required this.api});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  late Future<AccountStatus> _statusFuture;

  @override
  void initState() {
    super.initState();
    // widget.api already has username/driverId/role synchronously (set at
    // login), but the dispatcher's username is only known server-side (see
    // handleMe in backend/internal/httpapi/auth.go) - fetched fresh here
    // rather than cached on ApiClient since it can change any time the
    // dispatcher link is made/broken from the other side.
    _statusFuture = widget.api.fetchAccountStatus();
  }

  Future<void> _logout(BuildContext context) async {
    await clearSession(widget.api);
    if (!context.mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => LoginScreen(api: widget.api)),
      (route) => false,
    );
  }

  // Invalidates every token issued to this account (including the one this
  // very call used - see ApiClient.logoutAll), then logs out locally too,
  // same as _logout.
  Future<void> _logoutAll(BuildContext context) async {
    try {
      await widget.api.logoutAll();
    } catch (_) {
      // Non-fatal - even if the request fails, clearing the local session
      // below still logs this device out; other devices just keep working.
    }
    if (!context.mounted) return;
    await _logout(context);
  }

  void _changePassword(AccountStatus? status) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => ForgotPasswordScreen(api: widget.api, initialEmail: status?.email),
      ),
    );
  }

  String _roleLabel(String role) => role == 'dispatcher' ? 'Dispečer' : 'Vozač';

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Profil')),
      body: FutureBuilder<AccountStatus>(
        future: _statusFuture,
        builder: (context, snapshot) {
          final status = snapshot.data;
          return SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Row(
                          children: [
                            const CircleAvatar(
                              radius: 28,
                              backgroundColor: NocturneColors.accent800,
                              child: Icon(Icons.person, color: NocturneColors.accent300, size: 28),
                            ),
                            const SizedBox(width: 16),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(widget.api.username ?? 'Vozač', style: Theme.of(context).textTheme.titleMedium),
                                  Text(
                                    _roleLabel(status?.role ?? widget.api.role ?? 'driver'),
                                    style: Theme.of(context).textTheme.bodySmall,
                                  ),
                                ],
                              ),
                            ),
                            if (snapshot.connectionState == ConnectionState.waiting)
                              const SizedBox(
                                height: 16,
                                width: 16,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              ),
                          ],
                        ),
                        const Padding(
                          padding: EdgeInsets.symmetric(vertical: 16),
                          child: Divider(height: 1),
                        ),
                        _InfoRow(icon: Icons.badge_outlined, label: 'ID vozača', value: '${widget.api.driverId ?? status?.driverId ?? '-'}'),
                        _InfoRow(icon: Icons.account_circle_outlined, label: 'Korisničko ime', value: widget.api.username ?? '-'),
                        _InfoRow(
                          icon: Icons.email_outlined,
                          label: 'Email',
                          value: status?.email ?? 'Nije podešen',
                          trailing: status != null && status.email != null
                              ? Icon(
                                  status.emailVerified ? Icons.verified : Icons.error_outline,
                                  size: 18,
                                  color: status.emailVerified ? NocturneColors.accent300 : NocturneColors.warning,
                                )
                              : null,
                        ),
                        if (status?.dispatcherId != null)
                          _InfoRow(
                            icon: Icons.business_outlined,
                            label: 'Dispečer',
                            value: status?.dispatcherUsername ?? 'Dispečer #${status?.dispatcherId}',
                          ),
                        if (snapshot.hasError)
                          Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: Text(
                              'Neki podaci nisu mogli da se učitaju.',
                              style: TextStyle(color: NocturneColors.error, fontSize: 12),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 24),
                OutlinedButton.icon(
                  onPressed: () => _changePassword(status),
                  icon: const Icon(Icons.lock_reset),
                  label: const Text('Promeni lozinku'),
                ),
                const SizedBox(height: 8),
                OutlinedButton.icon(
                  onPressed: () => _logout(context),
                  icon: const Icon(Icons.logout),
                  label: const Text('Odjava'),
                ),
                const SizedBox(height: 8),
                OutlinedButton.icon(
                  onPressed: () => _logoutAll(context),
                  icon: const Icon(Icons.phonelink_erase),
                  label: const Text('Odjavi sve uređaje'),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final Widget? trailing;

  const _InfoRow({required this.icon, required this.label, required this.value, this.trailing});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        children: [
          Icon(icon, size: 20, color: NocturneColors.accent300),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: Theme.of(context).textTheme.bodySmall),
                Text(value, style: Theme.of(context).textTheme.bodyMedium),
              ],
            ),
          ),
          if (trailing != null) trailing!,
        ],
      ),
    );
  }
}
