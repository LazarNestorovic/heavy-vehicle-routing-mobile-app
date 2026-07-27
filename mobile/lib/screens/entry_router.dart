import 'package:flutter/material.dart';

import '../models/auth_result.dart';
import '../services/api_client.dart';
import '../services/auth_storage.dart';
import 'dispatcher_home_screen.dart';
import 'offered_trips_screen.dart';
import 'vehicle_list_screen.dart';

/// Applies a fresh AuthResult (from login, register, or Google sign-in) to
/// [api] and persists it via AuthStorage - shared so all three call sites
/// don't repeat the same field assignments.
Future<void> applySession(ApiClient api, AuthResult result) async {
  api.token = result.token;
  api.driverId = result.driverId;
  api.username = result.username;
  api.role = result.role;
  api.dispatcherId = result.dispatcherId;
  api.email = result.email;
  api.emailVerified = result.emailVerified;
  await AuthStorage().save(
    result.token,
    result.driverId,
    result.username,
    result.role,
    result.dispatcherId,
    email: result.email,
    emailVerified: result.emailVerified,
  );
}

/// Clears the local session (AuthStorage + every ApiClient field) - shared by
/// every screen's own "Odjava" action AND main.dart's global
/// ApiClient.onUnauthorized handler, which fires when the server rejects a
/// token (expired, or invalidated via "Odjavi sve uređaje" from another
/// device/session - see documentations/fixes/ entry).
Future<void> clearSession(ApiClient api) async {
  await AuthStorage().clear();
  api.token = null;
  api.driverId = null;
  api.username = null;
  api.role = null;
  api.dispatcherId = null;
  api.email = null;
  api.emailVerified = false;
}

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
