class RestStop {
  final double lat;
  final double lon;
  final String? name;
  final String amenity;

  const RestStop({required this.lat, required this.lon, this.name, required this.amenity});

  factory RestStop.fromJson(Map<String, dynamic> json) => RestStop(
        lat: (json['lat'] as num).toDouble(),
        lon: (json['lon'] as num).toDouble(),
        name: json['name'] as String?,
        amenity: json['amenity'] as String? ?? '',
      );

  String get label => (name != null && name!.isNotEmpty) ? name! : amenityLabel;

  String get amenityLabel {
    switch (amenity) {
      case 'fuel':
        return 'Benzinska pumpa';
      case 'parking':
        return 'Parking';
      case 'rest_area':
        return 'Odmaralište';
      default:
        return amenity;
    }
  }
}
