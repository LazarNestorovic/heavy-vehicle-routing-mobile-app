/// Response of GET /api/v1/vehicles/{id}/hours - a simplified stand-in for
/// real AETR driving-hours tracking (see backend/internal/store/trip.go
/// DrivingHours and documentations/features/2026-07-21-nocturne-redesign.md).
class VehicleHours {
  final double sinceLastBreakMin;
  final double drivingTodayMin;

  const VehicleHours({required this.sinceLastBreakMin, required this.drivingTodayMin});

  factory VehicleHours.fromJson(Map<String, dynamic> json) => VehicleHours(
        sinceLastBreakMin: (json['since_last_break_min'] as num).toDouble(),
        drivingTodayMin: (json['driving_today_min'] as num).toDouble(),
      );
}
