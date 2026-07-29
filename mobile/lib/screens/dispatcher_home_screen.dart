import 'package:flutter/material.dart';

import '../models/driver.dart';
import '../services/api_client.dart';
import '../theme/nocturne_theme.dart';
import '../widgets/email_verification_banner.dart';
import '../widgets/radial_fab_menu.dart';
import 'dispatcher_available_drivers_screen.dart';
import 'dispatcher_create_trip_screen.dart';
import 'dispatcher_live_map_screen.dart';
import 'dispatcher_trips_screen.dart';
import 'preferences_screen.dart';
import 'profile_screen.dart';
import 'vehicle_list_screen.dart';

/// Home screen for the dispatcher role - roster of managed drivers, fleet
/// management, and entry points to the live map and "find a driver" flow.
/// See documentations/features/ entry for the dispatcher/driver roles feature.
/// Profil/Preference/Vozila/Vozila uživo/Sve ture are reached via a
/// RadialFabMenu fixed on the left edge (non-draggable, hamburger icon) -
/// same pattern as VehicleListScreen/OfferedTripsScreen, for a consistent
/// navigation layout across the app. "Vozila" opens VehicleListScreen (reused
/// from the driver side - it's already fleet-aware, see its own doc comment)
/// to list/edit/delete the fleet, adding a new one from there instead of
/// jumping straight to a bare add form. "Odjava" lives on ProfileScreen, not
/// duplicated here; "Pronađi vozača" stays a plain, always-visible FAB since
/// it's this screen's primary action, not a secondary one.
class DispatcherHomeScreen extends StatefulWidget {
  final ApiClient api;
  const DispatcherHomeScreen({super.key, required this.api});

  @override
  State<DispatcherHomeScreen> createState() => _DispatcherHomeScreenState();
}

class _DispatcherHomeScreenState extends State<DispatcherHomeScreen> {
  late Future<List<Driver>> _driversFuture;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  void _reload() {
    setState(() {
      _driversFuture = widget.api.listManagedDrivers();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Moji vozači')),
      body: Stack(
        children: [
          Column(
            children: [
              EmailVerificationBanner(api: widget.api),
              Expanded(
                child: RefreshIndicator(
                  onRefresh: () async => _reload(),
                  child: FutureBuilder<List<Driver>>(
                    future: _driversFuture,
                    builder: (context, snapshot) {
                      if (snapshot.connectionState == ConnectionState.waiting) {
                        return const Center(child: CircularProgressIndicator());
                      }
                      if (snapshot.hasError) {
                        return Center(
                            child:
                                Text('Greška: ${snapshot.error}', style: const TextStyle(color: NocturneColors.error)));
                      }
                      final drivers = snapshot.data ?? [];
                      if (drivers.isEmpty) {
                        return ListView(
                          children: [
                            const Padding(
                              padding: EdgeInsets.all(32),
                              child: Center(child: Text('Nemaš još nijednog vozača u floti.')),
                            ),
                            Center(
                              child: FilledButton(
                                onPressed: () => Navigator.of(context).push(
                                  MaterialPageRoute(builder: (_) => DispatcherAvailableDriversScreen(api: widget.api)),
                                ),
                                child: const Text('Pronađi vozača'),
                              ),
                            ),
                          ],
                        );
                      }
                      return ListView.builder(
                        itemCount: drivers.length,
                        itemBuilder: (context, i) {
                          final d = drivers[i];
                          return ListTile(
                            leading: const Icon(Icons.person),
                            title: Text(d.username),
                            trailing: const Icon(Icons.chevron_right),
                            onTap: () => Navigator.of(context).push(
                              MaterialPageRoute(builder: (_) => DispatcherCreateTripScreen(api: widget.api, driver: d)),
                            ),
                          );
                        },
                      );
                    },
                  ),
                ),
              ),
            ],
          ),
          RadialFabMenu(
            initialCorner: FabCorner.bottomLeft,
            draggable: false,
            closedIcon: Icons.menu,
            items: [
              RadialFabMenuItem(
                icon: Icons.person_outline,
                tooltip: 'Profil',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => ProfileScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.tune,
                tooltip: 'Preference',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => PreferencesScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.local_shipping_outlined,
                tooltip: 'Vozila',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => VehicleListScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.map_outlined,
                tooltip: 'Vozila uživo',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => DispatcherLiveMapScreen(api: widget.api)),
                ),
              ),
              RadialFabMenuItem(
                icon: Icons.list_alt,
                tooltip: 'Sve ture',
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(builder: (_) => DispatcherTripsScreen(api: widget.api)),
                ),
              ),
            ],
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => Navigator.of(context).push(
          MaterialPageRoute(builder: (_) => DispatcherAvailableDriversScreen(api: widget.api)),
        ),
        tooltip: 'Pronađi vozača',
        child: const Icon(Icons.person_add),
      ),
    );
  }
}
