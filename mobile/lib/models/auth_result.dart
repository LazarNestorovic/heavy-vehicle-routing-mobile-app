class AuthResult {
  final String token;
  final int driverId;

  const AuthResult({required this.token, required this.driverId});

  factory AuthResult.fromJson(Map<String, dynamic> json) => AuthResult(
        token: json['token'] as String,
        driverId: json['driver_id'] as int,
      );
}
