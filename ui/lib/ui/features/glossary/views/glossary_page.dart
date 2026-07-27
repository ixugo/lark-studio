import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../view_models/glossary_view_model.dart';

class GlossaryPage extends HookConsumerWidget {
  const GlossaryPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final gs = ref.watch(glossaryProvider);
    final notifier = ref.read(glossaryProvider.notifier);
    final termController = useTextEditingController();
    final transController = useTextEditingController();

    useEffect(() {
      Future.microtask(() => notifier.load());
      return null;
    }, const []);

    Future<void> addTerm() async {
      final text = termController.text.trim();
      if (text.isEmpty) return;
      final trans = transController.text.trim();
      try {
        await notifier.addTerm(text, translation: trans);
        termController.clear();
        transController.clear();
      } catch (e) {
        if (context.mounted) _showToast(context, '添加失败: $e');
      }
    }

    Future<void> deleteTerm(int id) async {
      try {
        await notifier.deleteTerm(id);
      } catch (e) {
        if (context.mounted) _showToast(context, '删除失败: $e');
      }
    }

    return Column(
      children: [
        _buildToolbar(notifier),
        Expanded(
          child: gs.loading
              ? const Center(child: CupertinoActivityIndicator(radius: 14))
              : gs.error != null && gs.terms.isEmpty
                  ? Center(child: Text(gs.error!, style: const TextStyle(color: Color(0xFF8E8E93))))
                  : _buildContent(gs, termController, transController, addTerm, deleteTerm),
        ),
      ],
    );
  }

  Widget _buildToolbar(GlossaryNotifier notifier) {
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          height: 52,
          padding: const EdgeInsets.symmetric(horizontal: 28),
          decoration: BoxDecoration(color: const Color(0xFFFFFFFF).withValues(alpha: 0.65), border: const Border(bottom: BorderSide(color: Color(0x1A000000), width: 0.5))),
          child: Row(children: [
            const Text('词库', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: Color(0xFF1D1D1F), letterSpacing: -0.5)),
            const Spacer(),
            CupertinoButton(padding: EdgeInsets.zero, onPressed: () => notifier.load(), child: const Icon(CupertinoIcons.refresh, size: 16, color: Color(0xFF8E8E93))),
          ]),
        ),
      ),
    );
  }

  Widget _buildContent(GlossaryState gs, TextEditingController termController, TextEditingController transController, Future<void> Function() addTerm, Future<void> Function(int) deleteTerm) {
    return ListView(
      padding: const EdgeInsets.fromLTRB(28, 20, 28, 40),
      children: [
        _buildAddForm(termController, transController, addTerm),
        const SizedBox(height: 20),
        _buildTermList(gs, deleteTerm),
      ],
    );
  }

  Widget _buildAddForm(TextEditingController termController, TextEditingController transController, Future<void> Function() addTerm) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(14),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
        child: Container(
          padding: const EdgeInsets.all(18),
          decoration: BoxDecoration(
            gradient: LinearGradient(begin: Alignment.topLeft, end: Alignment.bottomRight, colors: [const Color(0xFFFFFFFF).withValues(alpha: 0.80), const Color(0xFFF9F9FB).withValues(alpha: 0.70)]),
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: const Color(0xFFFFFFFF).withValues(alpha: 0.5), width: 0.5),
            boxShadow: [BoxShadow(color: const Color(0xFF000000).withValues(alpha: 0.04), blurRadius: 14, offset: const Offset(0, 3))],
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const Text('添加术语', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
            const SizedBox(height: 4),
            const Text('翻译时保持原文不翻译的专有名词（不区分大小写）', style: TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
            const SizedBox(height: 14),
            Row(children: [
              Expanded(flex: 3, child: CupertinoTextField(controller: termController, placeholder: '源词 (如 Golang)', padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8), style: const TextStyle(fontSize: 13), onSubmitted: (_) => addTerm(), decoration: BoxDecoration(color: const Color(0xFF000000).withValues(alpha: 0.03), borderRadius: BorderRadius.circular(8), border: Border.all(color: const Color(0xFF000000).withValues(alpha: 0.06))))),
              const Padding(padding: EdgeInsets.symmetric(horizontal: 8), child: Icon(CupertinoIcons.arrow_right, size: 14, color: Color(0xFF8E8E93))),
              Expanded(flex: 3, child: CupertinoTextField(controller: transController, placeholder: '译文 (留空=保持原文)', padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8), style: const TextStyle(fontSize: 13), onSubmitted: (_) => addTerm(), decoration: BoxDecoration(color: const Color(0xFF000000).withValues(alpha: 0.03), borderRadius: BorderRadius.circular(8), border: Border.all(color: const Color(0xFF000000).withValues(alpha: 0.06))))),
              const SizedBox(width: 10),
              CupertinoButton(padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8), color: const Color(0xFF007AFF), borderRadius: BorderRadius.circular(8), minimumSize: Size.zero, onPressed: addTerm, child: const Text('添加', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: CupertinoColors.white))),
            ]),
          ]),
        ),
      ),
    );
  }

  Widget _buildTermList(GlossaryState gs, Future<void> Function(int) deleteTerm) {
    if (gs.terms.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(40),
        child: const Center(child: Column(children: [
          Icon(CupertinoIcons.book, size: 48, color: Color(0xFFC7C7CC)),
          SizedBox(height: 12),
          Text('暂无术语', style: TextStyle(fontSize: 15, color: Color(0xFF8E8E93))),
          SizedBox(height: 4),
          Text('添加后翻译时将保留原文', style: TextStyle(fontSize: 13, color: Color(0xFFC7C7CC))),
        ])),
      );
    }

    return ClipRRect(
      borderRadius: BorderRadius.circular(14),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
        child: Container(
          decoration: BoxDecoration(
            gradient: LinearGradient(begin: Alignment.topLeft, end: Alignment.bottomRight, colors: [const Color(0xFFFFFFFF).withValues(alpha: 0.80), const Color(0xFFF9F9FB).withValues(alpha: 0.70)]),
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: const Color(0xFFFFFFFF).withValues(alpha: 0.5), width: 0.5),
            boxShadow: [BoxShadow(color: const Color(0xFF000000).withValues(alpha: 0.04), blurRadius: 14, offset: const Offset(0, 3))],
          ),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Padding(padding: const EdgeInsets.fromLTRB(18, 16, 18, 8), child: Text('共 ${gs.terms.length} 个术语', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93)))),
            ...gs.terms.map((t) {
              final id = t['id'] as int;
              final text = t['text'] as String? ?? '';
              final trans = t['translation'] as String? ?? '';
              return _TermRow(text: text, translation: trans, onDelete: () => deleteTerm(id));
            }),
            const SizedBox(height: 8),
          ]),
        ),
      ),
    );
  }

  void _showToast(BuildContext context, String msg) {
    showCupertinoDialog(context: context, builder: (_) => CupertinoAlertDialog(content: Text(msg), actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))]));
  }
}

class _TermRow extends HookWidget {
  final String text;
  final String translation;
  final VoidCallback onDelete;
  const _TermRow({required this.text, required this.translation, required this.onDelete});

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final hasTranslation = translation.isNotEmpty && translation != text;

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
        decoration: BoxDecoration(
          color: hovering.value ? const Color(0xFF007AFF).withValues(alpha: 0.04) : CupertinoColors.transparent,
          border: const Border(bottom: BorderSide(color: Color(0x0A000000), width: 0.5)),
        ),
        child: Row(children: [
          Expanded(child: Row(children: [
            Text(text, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500, color: Color(0xFF1D1D1F))),
            if (hasTranslation) ...[
              const Padding(padding: EdgeInsets.symmetric(horizontal: 8), child: Icon(CupertinoIcons.arrow_right, size: 12, color: Color(0xFFC7C7CC))),
              Text(translation, style: const TextStyle(fontSize: 14, color: Color(0xFF007AFF))),
            ],
          ])),
          if (hovering.value) GestureDetector(onTap: onDelete, child: const Icon(CupertinoIcons.trash, size: 15, color: Color(0xFFFF3B30)))
          else const SizedBox(width: 15),
        ]),
      ),
    );
  }
}
