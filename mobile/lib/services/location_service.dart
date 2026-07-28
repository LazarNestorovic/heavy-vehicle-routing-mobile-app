import 'package:geolocator/geolocator.dart';

/// Why live GPS isn't active - surfaced to the driver (see
/// active_trip_screen.dart) instead of silently showing nothing with no
/// explanation. That silence was itself a reported bug: a driver whose
/// permission was denied (or denied forever from an earlier test, or whose
/// device location toggle is off) had no way to tell "GPS isn't working"
/// apart from "this app doesn't have real GPS at all" - see
/// documentations/fixes/ entry.
enum GpsStatus {
  granted,
  serviceDisabled, // the phone's location toggle is off
  denied, // just declined the (possibly just-shown) permission dialog
  deniedForever, // denied permanently - the OS won't show the dialog again
}

/// Wraps the geolocator plugin for live GPS tracking during an active trip
/// (see documentations/features/ live-GPS entry). Foreground-only - no
/// background location permission is requested, so tracking only runs while
/// ActiveTripScreen is open, matching this project's "keep permissions
/// minimal" approach and avoiding Play Store background-location review.
class LocationService {
  /// Requests location permission if needed - callers should tell the driver
  /// why for any non-[GpsStatus.granted] result, with a way to fix it (see
  /// [openLocationSettings]/[openAppSettings]).
  Future<GpsStatus> ensurePermission() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      return GpsStatus.serviceDisabled;
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }
    switch (permission) {
      case LocationPermission.whileInUse:
      case LocationPermission.always:
        return GpsStatus.granted;
      case LocationPermission.deniedForever:
        return GpsStatus.deniedForever;
      case LocationPermission.denied:
      case LocationPermission.unableToDetermine:
        return GpsStatus.denied;
    }
  }

  /// Position updates while this app is in the foreground, at most every
  /// [distanceFilterMeters] of movement (avoids flooding the backend with
  /// pings while stationary or moving slowly).
  Stream<Position> positionStream({int distanceFilterMeters = 20}) {
    return Geolocator.getPositionStream(
      locationSettings: LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: distanceFilterMeters),
    );
  }

  /// Opens the phone's location (GPS on/off) settings - for [GpsStatus.serviceDisabled].
  Future<void> openLocationSettings() => Geolocator.openLocationSettings();

  /// Opens this app's OS permission settings - for [GpsStatus.deniedForever],
  /// where the in-app permission dialog won't show again.
  Future<void> openAppSettings() => Geolocator.openAppSettings();

  /// One-shot current position, or null if permission isn't granted or the
  /// position can't be determined - used to prefill a route's origin with
  /// "where the driver already is" (see route_request_screen.dart).
  Future<Position?> getCurrentPosition() async {
    if (await ensurePermission() != GpsStatus.granted) return null;
    try {
      return await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(accuracy: LocationAccuracy.high),
      );
    } catch (_) {
      return null;
    }
  }

  /// One-shot distance (meters) from the phone's current position to
  /// (lat, lon) - used to gate actually STARTING a trip on being at its
  /// origin (see widgets/start_proximity_status.dart), mirroring turn-by-turn
  /// navigation apps. Returns null if permission isn't granted or the
  /// position can't be determined - callers should treat that as "can't
  /// verify" (offer a retry), not as "far away".
  Future<double?> distanceToCurrentPosition(double lat, double lon) async {
    final position = await getCurrentPosition();
    if (position == null) return null;
    return Geolocator.distanceBetween(position.latitude, position.longitude, lat, lon);
  }
}
