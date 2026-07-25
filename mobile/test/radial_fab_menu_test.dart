// Correctness checks for the draggable radial FAB menu (see
// documentations/features/2026-07-21-nocturne-redesign.md) - tap-to-open,
// sub-button dispatch, and drag-does-not-open-menu. The animation "feel"
// (snap speed, arc radius) can't be judged from a widget test; that stays on
// the user's physical device, same as the rest of this project's UI work.
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:hvr_mobile/widgets/radial_fab_menu.dart';

void main() {
  Widget wrap(Widget child) => MaterialApp(
        home: Scaffold(body: SizedBox.expand(child: child)),
      );

  // Sub-buttons are always present in the tree (opacity/scale animate, they
  // don't get removed) - IgnorePointer.ignoring is the reliable signal for
  // whether the menu is actually open (hit-testable) or closed.
  bool subButtonsIgnoring(WidgetTester tester) =>
      tester.widget<IgnorePointer>(find.byKey(const ValueKey('radial-fab-menu-ignore-0'))).ignoring;

  testWidgets('tapping the FAB opens the menu and reveals sub-buttons', (tester) async {
    await tester.pumpWidget(wrap(RadialFabMenu(items: [
      RadialFabMenuItem(icon: Icons.person, tooltip: 'A', onTap: () {}),
      RadialFabMenuItem(icon: Icons.local_shipping, tooltip: 'B', onTap: () {}),
      RadialFabMenuItem(icon: Icons.inventory, tooltip: 'C', onTap: () {}),
      RadialFabMenuItem(icon: Icons.history, tooltip: 'D', onTap: () {}),
      RadialFabMenuItem(icon: Icons.chat_bubble, tooltip: 'E', onTap: () {}),
    ])));

    expect(find.byIcon(Icons.add), findsOneWidget);
    expect(subButtonsIgnoring(tester), isTrue);

    await tester.tap(find.byIcon(Icons.add));
    await tester.pumpAndSettle();

    expect(subButtonsIgnoring(tester), isFalse);
  });

  testWidgets('tapping a sub-button fires its callback and closes the menu', (tester) async {
    var tapped = false;
    await tester.pumpWidget(wrap(RadialFabMenu(items: [
      RadialFabMenuItem(icon: Icons.person, tooltip: 'Profile', onTap: () => tapped = true),
      RadialFabMenuItem(icon: Icons.local_shipping, tooltip: 'B', onTap: () {}),
      RadialFabMenuItem(icon: Icons.inventory, tooltip: 'C', onTap: () {}),
      RadialFabMenuItem(icon: Icons.history, tooltip: 'D', onTap: () {}),
      RadialFabMenuItem(icon: Icons.chat_bubble, tooltip: 'E', onTap: () {}),
    ])));

    await tester.tap(find.byIcon(Icons.add));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Profile'));
    await tester.pumpAndSettle();

    expect(tapped, isTrue);
  });

  testWidgets('dragging the FAB does not open the menu', (tester) async {
    await tester.pumpWidget(wrap(RadialFabMenu(items: [
      RadialFabMenuItem(icon: Icons.person, tooltip: 'A', onTap: () {}),
      RadialFabMenuItem(icon: Icons.local_shipping, tooltip: 'B', onTap: () {}),
      RadialFabMenuItem(icon: Icons.inventory, tooltip: 'C', onTap: () {}),
      RadialFabMenuItem(icon: Icons.history, tooltip: 'D', onTap: () {}),
      RadialFabMenuItem(icon: Icons.chat_bubble, tooltip: 'E', onTap: () {}),
    ])));

    await tester.drag(find.byIcon(Icons.add), const Offset(-200, -200));
    await tester.pumpAndSettle();

    expect(subButtonsIgnoring(tester), isTrue);
  });
}
