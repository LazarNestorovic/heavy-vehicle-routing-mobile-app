/// One row of GET /api/v1/trips/{id}/events - the Trip Log timeline.
class TripEvent {
  final int id;
  final String eventType; // "departed" | "rest_stop_suggested" | "arrived"
  final String description;
  final DateTime occurredAt;

  const TripEvent({
    required this.id,
    required this.eventType,
    required this.description,
    required this.occurredAt,
  });

  factory TripEvent.fromJson(Map<String, dynamic> json) => TripEvent(
        id: json['id'] as int,
        eventType: json['event_type'] as String,
        description: json['description'] as String,
        occurredAt: DateTime.parse(json['occurred_at'] as String),
      );
}
