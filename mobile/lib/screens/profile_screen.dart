import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import 'entry_router.dart';
import 'login_screen.dart';

/// Mock-up's "Profile" panel: driver identity + sign out. No shift-tracking
/// backend exists (see truck_status_screen.dart for driving hours, which is
/// the closest real equivalent) - this stays intentionally simple.
class ProfileScreen extends StatelessWidget {
  final ApiClient api;
  const ProfileScreen({super.key, required this.api});

  Future<void> _logout(BuildContext context) async {
    await clearSession(api);
    if (!context.mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => LoginScreen(api: api)),
      (route) => false,
    );
  }

  // Invalidates every token issued to this account (including the one this
  // very call used - see ApiClient.logoutAll), then logs out locally too,
  // same as _logout.
  Future<void> _logoutAll(BuildContext context) async {
    try {
      await api.logoutAll();
    } catch (_) {
      // Non-fatal - even if the request fails, clearing the local session
      // below still logs this device out; other devices just keep working.
    }
    if (!context.mounted) return;
    await _logout(context);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Profil')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
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
                          Text(api.username ?? 'Vozač', style: Theme.of(context).textTheme.titleMedium),
                          if (api.driverId != null)
                            Text('ID: ${api.driverId}', style: Theme.of(context).textTheme.bodySmall),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 24),
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
      ),
    );
  }
}
