import 'package:flutter/material.dart';

import '../services/api_client.dart';
import 'dispatcher_home_screen.dart';
import 'offered_trips_screen.dart';
import 'vehicle_list_screen.dart';

/// Picks the right "home" screen after login/register/cold-start, based on
/// role and whether the driver is managed by a dispatcher (see
/// documentations/features/ entry for the dispatcher/driver roles feature):
///   - dispatcher -> DispatcherHomeScreen (fleet/roster management)
///   - managed driver (has a dispatcher) -> OfferedTripsScreen
///   - independent driver -> VehicleListScreen (unchanged, today's flow)
Widget homeScreenFor(ApiClient api) {
  if (api.role == 'dispatcher') {
    return DispatcherHomeScreen(api: api);
  }
  if (api.dispatcherId != null) {
    return OfferedTripsScreen(api: api);
  }
  return VehicleListScreen(api: api);
}
