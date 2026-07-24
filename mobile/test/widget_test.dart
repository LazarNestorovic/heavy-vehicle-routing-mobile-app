// Basic smoke test - verifies the app boots to the login screen when there's
// no stored token (the actual first screen now; see documentations/features/
// 2026-07-21-driver-preference-scoring.md for why auth is mandatory).
import 'package:flutter_test/flutter_test.dart';

import 'package:hvr_mobile/main.dart';
import 'package:hvr_mobile/services/api_client.dart';

void main() {
  testWidgets('App boots and shows the login screen when logged out', (WidgetTester tester) async {
    await tester.pumpWidget(HvrApp(api: ApiClient()));

    expect(find.text('Prijava'), findsOneWidget);
    expect(find.text('Prijavi se'), findsOneWidget);
  });
}
