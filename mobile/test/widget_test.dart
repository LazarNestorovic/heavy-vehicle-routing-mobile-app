// Basic smoke test - verifies the app boots and shows the vehicle profile
// screen (the actual first screen; the default `flutter create` template test
// this replaced tested a counter app that doesn't exist here).
import 'package:flutter_test/flutter_test.dart';

import 'package:hvr_mobile/main.dart';

void main() {
  testWidgets('App boots and shows the vehicle profile screen', (WidgetTester tester) async {
    await tester.pumpWidget(const HvrApp());

    expect(find.text('Profil vozila'), findsOneWidget);
    expect(find.text('Sačuvaj i nastavi'), findsOneWidget);
  });
}
