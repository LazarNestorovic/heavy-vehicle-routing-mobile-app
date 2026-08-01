/// One row of GET /api/v1/drivers - the "start a new chat" contact list.
/// Also reused for the dispatcher's managed/available driver lists, where
/// [email] may be present (used to disambiguate search-by-name/email
/// results - see documentations/features/ entry).
class Driver {
  final int id;
  final String username;
  final String? email;

  const Driver({required this.id, required this.username, this.email});

  factory Driver.fromJson(Map<String, dynamic> json) => Driver(
        id: json['id'] as int,
        username: json['username'] as String,
        email: json['email'] as String?,
      );
}
