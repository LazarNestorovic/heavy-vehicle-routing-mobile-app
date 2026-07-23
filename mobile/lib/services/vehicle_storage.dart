import 'package:shared_preferences/shared_preferences.dart';

import '../models/vehicle_profile.dart';

/// Persists the driver's vehicle profile locally so it doesn't need to be
/// re-entered on every app launch. No auth/multi-user support in this MVP
/// (see SPECIFIKACIJA.md) - one device, one vehicle profile.
class VehicleStorage {
  static const _keyId = 'vehicle_id';
  static const _keyHeight = 'vehicle_height_m';
  static const _keyWidth = 'vehicle_width_m';
  static const _keyLength = 'vehicle_length_m';
  static const _keyWeight = 'vehicle_weight_kg';
  static const _keyAxleLoad = 'vehicle_axle_load_kg';
  static const _keyHazmat = 'vehicle_hazmat';

  Future<VehicleProfile?> load() async {
    final prefs = await SharedPreferences.getInstance();
    final id = prefs.getInt(_keyId);
    final height = prefs.getDouble(_keyHeight);
    if (id == null || height == null) return null;

    return VehicleProfile(
      id: id,
      heightM: height,
      widthM: prefs.getDouble(_keyWidth) ?? 0,
      lengthM: prefs.getDouble(_keyLength) ?? 0,
      weightKg: prefs.getDouble(_keyWeight) ?? 0,
      axleLoadKg: prefs.getDouble(_keyAxleLoad) ?? 0,
      hazmat: prefs.getBool(_keyHazmat) ?? false,
    );
  }

  Future<void> save(VehicleProfile profile) async {
    final prefs = await SharedPreferences.getInstance();
    if (profile.id != null) await prefs.setInt(_keyId, profile.id!);
    await prefs.setDouble(_keyHeight, profile.heightM);
    await prefs.setDouble(_keyWidth, profile.widthM);
    await prefs.setDouble(_keyLength, profile.lengthM);
    await prefs.setDouble(_keyWeight, profile.weightKg);
    await prefs.setDouble(_keyAxleLoad, profile.axleLoadKg);
    await prefs.setBool(_keyHazmat, profile.hazmat);
  }

  Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyId);
    await prefs.remove(_keyHeight);
    await prefs.remove(_keyWidth);
    await prefs.remove(_keyLength);
    await prefs.remove(_keyWeight);
    await prefs.remove(_keyAxleLoad);
    await prefs.remove(_keyHazmat);
  }
}
