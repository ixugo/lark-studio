import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:vdub_ui/app.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('App renders', (WidgetTester tester) async {
    // 应用启动依赖本地持久化与窗口 channel，测试环境一律 mock
    SharedPreferences.setMockInitialValues({});
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
            const MethodChannel('vdub/window'), (call) async => null);

    await tester.pumpWidget(const ProviderScope(child: VdubApp()));
    await tester.pump();
    expect(find.byType(VdubApp), findsOneWidget);
  });
}
