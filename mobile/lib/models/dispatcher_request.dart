/// A pending (or resolved) dispatcher<->driver link request - see
/// documentations/features/ entry for the dispatcher/driver roles feature.
/// The dispatcher<->driver relationship is established only through this
/// flow (request + approval), never at registration.
class DispatcherRequest {
  final int id;
  final int dispatcherId;
  final String? dispatcherUsername;
  final int driverId;
  final String? driverUsername;
  final String status; // "pending" | "approved" | "rejected"
  final DateTime createdAt;

  const DispatcherRequest({
    required this.id,
    required this.dispatcherId,
    this.dispatcherUsername,
    required this.driverId,
    this.driverUsername,
    required this.status,
    required this.createdAt,
  });

  factory DispatcherRequest.fromJson(Map<String, dynamic> json) => DispatcherRequest(
        id: json['id'] as int,
        dispatcherId: json['dispatcher_id'] as int,
        dispatcherUsername: json['dispatcher_username'] as String?,
        driverId: json['driver_id'] as int,
        driverUsername: json['driver_username'] as String?,
        status: json['status'] as String,
        createdAt: DateTime.parse(json['created_at'] as String),
      );
}
