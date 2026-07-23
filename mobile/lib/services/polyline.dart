import 'package:latlong2/latlong.dart';

/// Decodes Valhalla's default shape encoding (Google's polyline algorithm at
/// 1e-6 precision, "polyline6"). Mirrors backend/internal/valhalla/polyline.go -
/// keep both in sync if the encoding ever changes.
List<LatLng> decodePolyline6(String encoded) {
  final List<LatLng> coords = [];
  int index = 0, lat = 0, lon = 0;

  while (index < encoded.length) {
    int shift = 0, result = 0, b;
    do {
      b = encoded.codeUnitAt(index) - 63;
      index++;
      result |= (b & 0x1f) << shift;
      shift += 5;
    } while (b >= 0x20);
    lat += (result & 1) != 0 ? ~(result >> 1) : (result >> 1);

    shift = 0;
    result = 0;
    do {
      b = encoded.codeUnitAt(index) - 63;
      index++;
      result |= (b & 0x1f) << shift;
      shift += 5;
    } while (b >= 0x20);
    lon += (result & 1) != 0 ? ~(result >> 1) : (result >> 1);

    coords.add(LatLng(lat / 1e6, lon / 1e6));
  }
  return coords;
}
