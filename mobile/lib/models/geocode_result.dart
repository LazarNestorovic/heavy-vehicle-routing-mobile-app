/// One match from GET /api/v1/geocode (Nominatim proxy - see
/// documentations/features/ entry for why address search went through
/// Nominatim rather than Google's paid Geocoding API).
class GeocodeResult {
  final double lat;
  final double lon;
  final String displayName;

  const GeocodeResult({required this.lat, required this.lon, required this.displayName});

  factory GeocodeResult.fromJson(Map<String, dynamic> json) => GeocodeResult(
        lat: (json['lat'] as num).toDouble(),
        lon: (json['lon'] as num).toDouble(),
        displayName: json['display_name'] as String,
      );
}
