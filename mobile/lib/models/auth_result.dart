class AuthResult {
  final String token;
  final int driverId;
  final String role;
  final int? dispatcherId;

  const AuthResult({required this.token, required this.driverId, required this.role, this.dispatcherId});

  factory AuthResult.fromJson(Map<String, dynamic> json) => AuthResult(
        token: json['token'] as String,
        driverId: json['driver_id'] as int,
        role: json['role'] as String,
        dispatcherId: json['dispatcher_id'] as int?,
      );
}
