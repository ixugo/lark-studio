import 'package:flutter_test/flutter_test.dart';
import 'package:vdub_ui/app.dart';

void main() {
  testWidgets('App renders', (WidgetTester tester) async {
    await tester.pumpWidget(const VdubApp());
    await tester.pump();
  });
}
