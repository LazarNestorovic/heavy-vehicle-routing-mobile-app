/// One row of GET /api/v1/drivers - the "start a new chat" contact list.
class Driver {
  final int id;
  final String username;

  const Driver({required this.id, required this.username});

  factory Driver.fromJson(Map<String, dynamic> json) => Driver(
        id: json['id'] as int,
        username: json['username'] as String,
      );
}
