/// Response of GET /api/v1/auth/me - the caller's current account state,
/// fetched on demand (not cached at login) so it reflects changes that
/// happened outside the app (email verified via browser link, dispatcher
/// link removed by the other side) without requiring a full re-login. See
/// documentations/fixes/2026-07-26-*.md entries.
class AccountStatus {
  final int driverId;
  final String username;
  final String role;
  final int? dispatcherId;
  final String? dispatcherUsername;
  final String? email;
  final bool emailVerified;

  const AccountStatus({
    required this.driverId,
    required this.username,
    required this.role,
    this.dispatcherId,
    this.dispatcherUsername,
    this.email,
    required this.emailVerified,
  });

  factory AccountStatus.fromJson(Map<String, dynamic> json) => AccountStatus(
        driverId: json['driver_id'] as int,
        username: json['username'] as String,
        role: json['role'] as String,
        dispatcherId: json['dispatcher_id'] as int?,
        dispatcherUsername: json['dispatcher_username'] as String?,
        email: json['email'] as String?,
        emailVerified: json['email_verified'] as bool? ?? false,
      );
}
