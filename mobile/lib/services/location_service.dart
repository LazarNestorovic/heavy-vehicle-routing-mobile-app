import 'package:geolocator/geolocator.dart';

/// Wraps the geolocator plugin for live GPS tracking during an active trip
/// (see documentations/features/ live-GPS entry). Foreground-only - no
/// background location permission is requested, so tracking only runs while
/// ActiveTripScreen is open, matching this project's "keep permissions
/// minimal" approach and avoiding Play Store background-location review.
class LocationService {
  /// Requests location permission if needed. Returns false if the user
  /// denies it (or denied it permanently) - callers should fall back to the
  /// existing simulated WS playback in that case, not treat it as an error.
  Future<bool> ensurePermission() async {
    if (!await Geolocator.isLocationServiceEnabled()) return false;

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }
    return permission == LocationPermission.whileInUse || permission == LocationPermission.always;
  }

  /// Position updates while this app is in the foreground, at most every
  /// [distanceFilterMeters] of movement (avoids flooding the backend with
  /// pings while stationary or moving slowly).
  Stream<Position> positionStream({int distanceFilterMeters = 20}) {
    return Geolocator.getPositionStream(
      locationSettings: LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: distanceFilterMeters),
    );
  }
}
