/// A driver's saved favorite stop (backend/internal/store/favorite_stop.go) -
/// distinct from RestStop, which is a worker-computed rest suggestion. This one
/// is something the driver explicitly saved, no `amenity` field.
class FavoriteStop {
  final int id;
  final double lat;
  final double lon;
  final String name;

  const FavoriteStop({required this.id, required this.lat, required this.lon, required this.name});

  factory FavoriteStop.fromJson(Map<String, dynamic> json) => FavoriteStop(
        id: json['id'] as int,
        lat: (json['lat'] as num).toDouble(),
        lon: (json['lon'] as num).toDouble(),
        name: json['name'] as String? ?? '',
      );
}
