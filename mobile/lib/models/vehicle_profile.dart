class VehicleProfile {
  final int? id;
  final double heightM;
  final double widthM;
  final double lengthM;
  final double weightKg;
  final double axleLoadKg;
  final bool hazmat;
  final double fuelPercent;
  final double? nextServiceKm;

  const VehicleProfile({
    this.id,
    required this.heightM,
    required this.widthM,
    required this.lengthM,
    required this.weightKg,
    required this.axleLoadKg,
    required this.hazmat,
    this.fuelPercent = 100,
    this.nextServiceKm,
  });

  Map<String, dynamic> toJson() => {
        'height_m': heightM,
        'width_m': widthM,
        'length_m': lengthM,
        'weight_kg': weightKg,
        'axle_load_kg': axleLoadKg,
        'hazmat': hazmat,
      };

  factory VehicleProfile.fromJson(Map<String, dynamic> json) => VehicleProfile(
        id: json['id'] as int?,
        heightM: (json['height_m'] as num).toDouble(),
        widthM: (json['width_m'] as num).toDouble(),
        lengthM: (json['length_m'] as num).toDouble(),
        weightKg: (json['weight_kg'] as num).toDouble(),
        axleLoadKg: (json['axle_load_kg'] as num).toDouble(),
        hazmat: json['hazmat'] as bool,
        fuelPercent: (json['fuel_percent'] as num?)?.toDouble() ?? 100,
        nextServiceKm: (json['next_service_km'] as num?)?.toDouble(),
      );

  VehicleProfile copyWith({int? id}) => VehicleProfile(
        id: id ?? this.id,
        heightM: heightM,
        widthM: widthM,
        lengthM: lengthM,
        weightKg: weightKg,
        axleLoadKg: axleLoadKg,
        hazmat: hazmat,
        fuelPercent: fuelPercent,
        nextServiceKm: nextServiceKm,
      );
}
