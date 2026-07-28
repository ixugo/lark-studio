import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:vdub_ui/ui/core/glass_card.dart';

/// main 验证玻璃卡片在 macOS 主题下可稳定渲染内容。
void main() {
  testWidgets('GlassCard 应渲染其子组件', (tester) async {
    await tester.pumpWidget(
      MacosApp(
        home: MacosWindow(
          child: Center(
            child: SizedBox(
              width: 300,
              height: 200,
              child: GlassCard(child: const Center(child: Text('hello-glass'))),
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('hello-glass'), findsOneWidget);
  });
}
