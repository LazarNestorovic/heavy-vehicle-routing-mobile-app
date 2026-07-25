import 'package:flutter/material.dart';

import '../services/api_client.dart';
import '../services/auth_storage.dart';
import '../theme/nocturne_theme.dart';
import 'login_screen.dart';

/// Mock-up's "Profile" panel: driver identity + sign out. No shift-tracking
/// backend exists (see truck_status_screen.dart for driving hours, which is
/// the closest real equivalent) - this stays intentionally simple.
class ProfileScreen extends StatelessWidget {
  final ApiClient api;
  const ProfileScreen({super.key, required this.api});

  Future<void> _logout(BuildContext context) async {
    await AuthStorage().clear();
    api.token = null;
    api.driverId = null;
    api.username = null;
    if (!context.mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => LoginScreen(api: api)),
      (route) => false,
    );
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
          ],
        ),
      ),
    );
  }
}
