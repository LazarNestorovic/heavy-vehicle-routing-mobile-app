/// Mirrors backend/internal/store/preferences.go DriverPreferences. 1-5 scale,
/// 3 is "neutral" (see backend default).
class DriverPreferences {
  final int fuelPriority;
  final int cargoPriority;
  final int highwayPriority;
  final int timePriority;
  final String? preferredFuelBrand;

  const DriverPreferences({
    required this.fuelPriority,
    required this.cargoPriority,
    required this.highwayPriority,
    required this.timePriority,
    this.preferredFuelBrand,
  });

  static const neutral = DriverPreferences(
    fuelPriority: 3, cargoPriority: 3, highwayPriority: 3, timePriority: 3,
  );

  Map<String, dynamic> toJson() => {
        'fuel_priority': fuelPriority,
        'cargo_priority': cargoPriority,
        'highway_priority': highwayPriority,
        'time_priority': timePriority,
        if (preferredFuelBrand != null && preferredFuelBrand!.isNotEmpty)
          'preferred_fuel_brand': preferredFuelBrand,
      };

  factory DriverPreferences.fromJson(Map<String, dynamic> json) => DriverPreferences(
        fuelPriority: json['fuel_priority'] as int,
        cargoPriority: json['cargo_priority'] as int,
        highwayPriority: json['highway_priority'] as int,
        timePriority: json['time_priority'] as int,
        preferredFuelBrand: json['preferred_fuel_brand'] as String?,
      );

  DriverPreferences copyWith({
    int? fuelPriority,
    int? cargoPriority,
    int? highwayPriority,
    int? timePriority,
    String? preferredFuelBrand,
  }) =>
      DriverPreferences(
        fuelPriority: fuelPriority ?? this.fuelPriority,
        cargoPriority: cargoPriority ?? this.cargoPriority,
        highwayPriority: highwayPriority ?? this.highwayPriority,
        timePriority: timePriority ?? this.timePriority,
        preferredFuelBrand: preferredFuelBrand ?? this.preferredFuelBrand,
      );
}
