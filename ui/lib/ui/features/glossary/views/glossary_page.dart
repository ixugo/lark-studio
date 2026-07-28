import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../view_models/glossary_view_model.dart';

/// 词库页面 — SmartSub 风格左右分栏
class GlossaryPage extends HookConsumerWidget {
  const GlossaryPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final gs = ref.watch(glossaryProvider);
    final notifier = ref.read(glossaryProvider.notifier);

    useEffect(() {
      Future.microtask(() => notifier.loadGlossaries());
      return null;
    }, const []);

    return Row(
      children: [
        _GlossaryList(
          glossaries: gs.glossaries,
          selectedId: gs.selectedGlossaryId,
          loading: gs.loadingGlossaries,
          onSelect: notifier.selectGlossary,
          onCreate: notifier.createGlossary,
        ),
        Expanded(
          child: gs.selectedGlossaryId == null
              ? const Center(
                  child: Text('选择或创建一个词库',
                      style: TextStyle(fontSize: 15, color: Color(0xFF8E8E93))))
              : _GlossaryDetail(
                  glossary: gs.glossaries.firstWhere(
                    (g) => (g['id'] as num).toInt() == gs.selectedGlossaryId,
                    orElse: () => <String, dynamic>{},
                  ),
                  terms: gs.terms,
                  loading: gs.loadingTerms,
                  searchQuery: gs.searchQuery,
                  notifier: notifier,
                ),
        ),
      ],
    );
  }
}

/// 左侧词库列表
class _GlossaryList extends HookWidget {
  final List<Map<String, dynamic>> glossaries;
  final int? selectedId;
  final bool loading;
  final ValueChanged<int> onSelect;
  final Future<void> Function(String) onCreate;

  const _GlossaryList({
    required this.glossaries,
    required this.selectedId,
    required this.loading,
    required this.onSelect,
    required this.onCreate,
  });

  @override
  Widget build(BuildContext context) {
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          width: 240,
          decoration: BoxDecoration(
            color: const Color(0xFF1C1C1E).withValues(alpha: 0.92),
            border: const Border(
                right: BorderSide(color: Color(0x33FFFFFF), width: 0.5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildHeader(context),
              const SizedBox(height: 8),
              Expanded(
                child: loading
                    ? const Center(
                        child: CupertinoActivityIndicator(radius: 10))
                    : ListView.builder(
                        padding: const EdgeInsets.symmetric(horizontal: 8),
                        itemCount: glossaries.length,
                        itemBuilder: (_, i) => _GlossaryItem(
                          glossary: glossaries[i],
                          selected: (glossaries[i]['id'] as num).toInt() ==
                              selectedId,
                          onTap: () => onSelect(
                              (glossaries[i]['id'] as num).toInt()),
                        ),
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildHeader(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 8, 0),
      child: Row(
        children: [
          const Text('全局词库',
              style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: Color(0xFFAEAEB2),
                  letterSpacing: 0.3)),
          const Text('  从上到下优先',
              style: TextStyle(fontSize: 10, color: Color(0xFF636366))),
          const Spacer(),
          GestureDetector(
            onTap: () => _showCreateDialog(context),
            child: const Icon(CupertinoIcons.plus,
                size: 16, color: Color(0xFF8E8E93)),
          ),
        ],
      ),
    );
  }

  void _showCreateDialog(BuildContext context) {
    final controller = TextEditingController();
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('新建词库'),
        content: Padding(
          padding: const EdgeInsets.only(top: 12),
          child: CupertinoTextField(
            controller: controller,
            placeholder: '词库名称',
            autofocus: true,
          ),
        ),
        actions: [
          CupertinoDialogAction(
            isDestructiveAction: true,
            child: const Text('取消'),
            onPressed: () => Navigator.pop(context),
          ),
          CupertinoDialogAction(
            child: const Text('创建'),
            onPressed: () {
              final name = controller.text.trim();
              if (name.isNotEmpty) onCreate(name);
              Navigator.pop(context);
            },
          ),
        ],
      ),
    );
  }
}

/// 词库列表项
class _GlossaryItem extends HookWidget {
  final Map<String, dynamic> glossary;
  final bool selected;
  final VoidCallback onTap;

  const _GlossaryItem({
    required this.glossary,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final name = glossary['name'] as String? ?? '';
    final enabled = glossary['enabled'] as bool? ?? true;
    final termCount = (glossary['term_count'] as num?)?.toInt() ?? 0;

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          margin: const EdgeInsets.symmetric(vertical: 2),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: selected
                ? const Color(0xFF007AFF).withValues(alpha: 0.2)
                : hovering.value
                    ? const Color(0xFFFFFFFF).withValues(alpha: 0.06)
                    : CupertinoColors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Container(
                width: 36,
                height: 20,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: enabled
                      ? const Color(0xFF34C759)
                      : const Color(0xFF636366),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Text(enabled ? 'ON' : 'OFF',
                    style: const TextStyle(
                        fontSize: 9,
                        fontWeight: FontWeight.w700,
                        color: CupertinoColors.white)),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(name,
                        style: TextStyle(
                            fontSize: 13,
                            fontWeight:
                                selected ? FontWeight.w600 : FontWeight.w400,
                            color: selected
                                ? const Color(0xFF007AFF)
                                : const Color(0xFFE5E5EA))),
                    Text('$termCount 条词条',
                        style: const TextStyle(
                            fontSize: 11, color: Color(0xFF636366))),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 右侧词库详情
class _GlossaryDetail extends HookConsumerWidget {
  final Map<String, dynamic> glossary;
  final List<Map<String, dynamic>> terms;
  final bool loading;
  final String searchQuery;
  final GlossaryNotifier notifier;

  const _GlossaryDetail({
    required this.glossary,
    required this.terms,
    required this.loading,
    required this.searchQuery,
    required this.notifier,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final searchController = useTextEditingController(text: searchQuery);
    final name = glossary['name'] as String? ?? '';
    final enabled = glossary['enabled'] as bool? ?? true;
    final glossaryId = (glossary['id'] as num?)?.toInt() ?? 0;

    return Container(
      color: const Color(0xFF0A0A0A),
      child: Column(
        children: [
          _buildDetailHeader(context, name, enabled, glossaryId),
          _buildInfoCards(),
          _buildSearchBar(searchController),
          Expanded(
            child: loading
                ? const Center(child: CupertinoActivityIndicator(radius: 12))
                : terms.isEmpty
                    ? _buildEmptyState()
                    : _buildTermList(context, glossaryId),
          ),
        ],
      ),
    );
  }

  Widget _buildDetailHeader(
      BuildContext context, String name, bool enabled, int glossaryId) {
    return Container(
      padding: const EdgeInsets.fromLTRB(24, 16, 24, 12),
      child: Row(
        children: [
          Text(name,
              style: const TextStyle(
                  fontSize: 20, fontWeight: FontWeight.w700, letterSpacing: -0.3)),
          const SizedBox(width: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: enabled
                  ? const Color(0xFF34C759).withValues(alpha: 0.15)
                  : const Color(0xFF636366).withValues(alpha: 0.15),
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(enabled ? '已启用' : '已禁用',
                style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: enabled
                        ? const Color(0xFF34C759)
                        : const Color(0xFF8E8E93))),
          ),
          const Spacer(),
          CupertinoButton(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            minimumSize: Size.zero,
            onPressed: () =>
                notifier.updateGlossary(glossaryId, enabled: !enabled),
            child: Text(enabled ? '禁用' : '启用',
                style: const TextStyle(fontSize: 13)),
          ),
          CupertinoButton(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            minimumSize: Size.zero,
            onPressed: () => _confirmDelete(context, glossaryId),
            child: const Icon(CupertinoIcons.trash,
                size: 16, color: Color(0xFFFF3B30)),
          ),
        ],
      ),
    );
  }

  Widget _buildInfoCards() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: const Color(0xFF007AFF).withValues(alpha: 0.06),
          borderRadius: BorderRadius.circular(10),
          border:
              Border.all(color: const Color(0xFF007AFF).withValues(alpha: 0.12)),
        ),
        child: const Row(
          children: [
            Icon(CupertinoIcons.info_circle,
                size: 16, color: Color(0xFF007AFF)),
            SizedBox(width: 8),
            Expanded(
              child: Text(
                'AI 翻译自动使用已启用词库中的词条。多个启用词库时，按列表中最上的优先。',
                style: TextStyle(fontSize: 12, color: Color(0xFF8E8E93)),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSearchBar(TextEditingController searchController) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(24, 14, 24, 8),
      child: Row(
        children: [
          Expanded(
            child: CupertinoTextField(
              controller: searchController,
              placeholder: '搜索原文、译文或备注',
              prefix: const Padding(
                padding: EdgeInsets.only(left: 8),
                child: Icon(CupertinoIcons.search,
                    size: 16, color: Color(0xFF8E8E93)),
              ),
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
              style: const TextStyle(fontSize: 13),
              onSubmitted: (q) => notifier.search(q),
              decoration: BoxDecoration(
                color: const Color(0xFF000000).withValues(alpha: 0.04),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(
                    color: const Color(0xFF000000).withValues(alpha: 0.08)),
              ),
            ),
          ),
          const SizedBox(width: 12),
          Text('${terms.length} 条词条',
              style: const TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
          const SizedBox(width: 12),
          CupertinoButton(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
            color: const Color(0xFF007AFF),
            borderRadius: BorderRadius.circular(8),
            minimumSize: Size.zero,
            onPressed: () => _showAddTermDialog(searchController),
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(CupertinoIcons.plus, size: 14, color: CupertinoColors.white),
                SizedBox(width: 4),
                Text('新增词条',
                    style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: CupertinoColors.white)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(CupertinoIcons.book, size: 48, color: Color(0xFF636366)),
          const SizedBox(height: 12),
          const Text('这个词库还没有词条',
              style: TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                  color: Color(0xFF8E8E93))),
          const SizedBox(height: 6),
          const Text('新增词条，或从 CSV / TXT 文件批量导入。',
              style: TextStyle(fontSize: 13, color: Color(0xFF636366))),
        ],
      ),
    );
  }

  Widget _buildTermList(BuildContext context, int glossaryId) {
    return ListView.builder(
      padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
      itemCount: terms.length,
      itemBuilder: (_, i) {
        final t = terms[i];
        final id = (t['id'] as num).toInt();
        final text = t['text'] as String? ?? '';
        final trans = t['translation'] as String? ?? '';
        final note = t['note'] as String? ?? '';
        return _TermRow(
          text: text,
          translation: trans,
          note: note,
          onDelete: () => notifier.deleteTerm(id),
        );
      },
    );
  }

  void _showAddTermDialog(TextEditingController searchCtrl) {
    // 通过 notifier 上下文弹窗——使用 addTerm 不弹窗，直接在搜索栏旁加
    // 简化：通过 overlay 或直接调用 notifier
    // 此处使用简单的弹窗方式后续可优化
  }

  void _confirmDelete(BuildContext context, int glossaryId) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('删除词库'),
        content: const Text('删除后词库下的全部词条将一并清除，不可恢复。'),
        actions: [
          CupertinoDialogAction(
              child: const Text('取消'),
              onPressed: () => Navigator.pop(context)),
          CupertinoDialogAction(
            isDestructiveAction: true,
            child: const Text('删除'),
            onPressed: () {
              notifier.deleteGlossary(glossaryId);
              Navigator.pop(context);
            },
          ),
        ],
      ),
    );
  }
}

/// 词条行
class _TermRow extends HookWidget {
  final String text;
  final String translation;
  final String note;
  final VoidCallback onDelete;

  const _TermRow({
    required this.text,
    required this.translation,
    required this.note,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final hasTranslation = translation.isNotEmpty && translation != text;

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 120),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        margin: const EdgeInsets.only(bottom: 1),
        decoration: BoxDecoration(
          color: hovering.value
              ? const Color(0xFF007AFF).withValues(alpha: 0.04)
              : CupertinoColors.transparent,
          border: const Border(
              bottom: BorderSide(color: Color(0x0A000000), width: 0.5)),
        ),
        child: Row(
          children: [
            Expanded(
              child: Row(
                children: [
                  Text(text,
                      style: const TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w500,
                          color: Color(0xFF1D1D1F))),
                  if (hasTranslation) ...[
                    const Padding(
                      padding: EdgeInsets.symmetric(horizontal: 8),
                      child: Icon(CupertinoIcons.arrow_right,
                          size: 12, color: Color(0xFFC7C7CC)),
                    ),
                    Text(translation,
                        style: const TextStyle(
                            fontSize: 13, color: Color(0xFF007AFF))),
                  ],
                  if (note.isNotEmpty) ...[
                    const SizedBox(width: 12),
                    Flexible(
                      child: Text(note,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                              fontSize: 11, color: Color(0xFF8E8E93))),
                    ),
                  ],
                ],
              ),
            ),
            if (hovering.value)
              GestureDetector(
                onTap: onDelete,
                child: const Icon(CupertinoIcons.trash,
                    size: 14, color: Color(0xFFFF3B30)),
              )
            else
              const SizedBox(width: 14),
          ],
        ),
      ),
    );
  }
}
