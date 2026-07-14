import 'package:coderoam/main.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows CodeRoam touch spike shell', (tester) async {
    await tester.pumpWidget(const CodeRoamApp());
    expect(find.text('CodeRoam'), findsOneWidget);
    expect(find.text('Editor touch spike'), findsOneWidget);
  });
}
