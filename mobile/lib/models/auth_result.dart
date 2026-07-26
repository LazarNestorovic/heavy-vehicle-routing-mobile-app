class AuthResult {
  final String token;
  final int driverId;
  final String username;
  final String role;
  final int? dispatcherId;
  final String? email;
  final bool emailVerified;

  const AuthResult({
    required this.token,
    required this.driverId,
    required this.username,
    required this.role,
    this.dispatcherId,
    this.email,
    this.emailVerified = false,
  });

  factory AuthResult.fromJson(Map<String, dynamic> json) => AuthResult(
        token: json['token'] as String,
        driverId: json['driver_id'] as int,
        username: json['username'] as String,
        role: json['role'] as String,
        dispatcherId: json['dispatcher_id'] as int?,
        email: json['email'] as String?,
        emailVerified: json['email_verified'] as bool? ?? false,
      );
}
