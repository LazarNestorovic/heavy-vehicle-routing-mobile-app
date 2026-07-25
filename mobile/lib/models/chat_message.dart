/// One message, from GET /api/v1/chats/{driverId}/messages, POST of the same,
/// or a live push over GET /ws/chats/{counterpartId}.
class ChatMessage {
  final int id;
  final int fromDriverId;
  final int toDriverId;
  final String body;
  final DateTime sentAt;
  final DateTime? readAt;

  const ChatMessage({
    required this.id,
    required this.fromDriverId,
    required this.toDriverId,
    required this.body,
    required this.sentAt,
    this.readAt,
  });

  factory ChatMessage.fromJson(Map<String, dynamic> json) => ChatMessage(
        id: json['id'] as int,
        fromDriverId: json['from_driver_id'] as int,
        toDriverId: json['to_driver_id'] as int,
        body: json['body'] as String,
        sentAt: DateTime.parse(json['sent_at'] as String),
        readAt: json['read_at'] != null ? DateTime.parse(json['read_at'] as String) : null,
      );
}

/// One row of GET /api/v1/chats - the chat list screen's conversation summary.
class ChatConversation {
  final int counterpartId;
  final String lastMessage;
  final DateTime lastMessageAt;
  final int unreadCount;

  const ChatConversation({
    required this.counterpartId,
    required this.lastMessage,
    required this.lastMessageAt,
    required this.unreadCount,
  });

  factory ChatConversation.fromJson(Map<String, dynamic> json) => ChatConversation(
        counterpartId: json['counterpart_id'] as int,
        lastMessage: json['last_message'] as String,
        lastMessageAt: DateTime.parse(json['last_message_at'] as String),
        unreadCount: json['unread_count'] as int,
      );
}
