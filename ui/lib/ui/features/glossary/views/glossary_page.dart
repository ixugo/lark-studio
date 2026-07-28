import 'dart:ui';
import 'package:flutter/cupertino.dart'
    show
        CupertinoAlertDialog,
        CupertinoDialogAction,
        CupertinoIcons,
        showCupertinoDialog;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../core/app_colors.dart';
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

    final c = AppColors.of(context);
    return Row(
      children: [
        _GlossaryList(
          glossaries: gs.glossaries,
          selectedId: gs.selectedGlossaryId,
          loading: gs.loadingGlossaries,
          onSelect: notifier.selectGlossary,
          onCreate: notifier.createGlossary,
          onDelete: notifier.deleteGlossary,
          onToggle: (id, enabled) =>
              notifier.updateGlossary(id, enabled: enabled),
        ),
        Expanded(
          child: gs.selectedGlossaryId == null
              ? Center(
                  child: Text(
                    '选择或创建一个词库',
                    style: TextStyle(fontSize: 15, color: c.textSecondary),
                  ),
                )
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
  final Future<void> Function(int) onDelete;
  final Future<void> Function(int, bool) onToggle;

  const _GlossaryList({
    required this.glossaries,
    required this.selectedId,
    required this.loading,
    required this.onSelect,
    required this.onCreate,
    required this.onDelete,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    final adding = useState(false);
    final addController = useTextEditingController();
    final c = AppColors.of(context);

    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          width: 240,
          decoration: BoxDecoration(
            color: c.barBg.withValues(alpha: 0.92),
            border: Border(right: BorderSide(color: c.borderLight, width: 0.5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildHeader(context, adding, c),
              const SizedBox(height: 8),
              if (adding.value)
                Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 4,
                  ),
                  child: MacosTextField(
                    controller: addController,
                    placeholder: '输入词库名称，回车确认',
                    autofocus: true,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 8,
                    ),
                    decoration: BoxDecoration(
                      color: c.inputBg,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    style: TextStyle(fontSize: 12, color: c.textPrimary),
                    placeholderStyle: TextStyle(
                      fontSize: 12,
                      color: c.textPlaceholder,
                    ),
                    onSubmitted: (value) async {
                      final name = value.trim();
                      if (name.isNotEmpty) {
                        await onCreate(name);
                      }
                      addController.clear();
                      adding.value = false;
                    },
                  ),
                ),
              Expanded(
                child: loading
                    ? const Center(child: ProgressCircle(radius: 10))
                    : ListView.builder(
                        padding: const EdgeInsets.symmetric(horizontal: 8),
                        itemCount: glossaries.length,
                        itemBuilder: (_, i) {
                          final id = (glossaries[i]['id'] as num).toInt();
                          return _GlossaryItem(
                            glossary: glossaries[i],
                            selected: id == selectedId,
                            onTap: () => onSelect(id),
                            onDelete: () => onDelete(id),
                            onToggle: (value) => onToggle(id, value),
                          );
                        },
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildHeader(
    BuildContext context,
    ValueNotifier<bool> adding,
    AppColors c,
  ) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 8, 0),
      child: Row(
        children: [
          Text(
            '词库',
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: c.textPrimary,
              letterSpacing: 0.3,
            ),
          ),
          const Spacer(),
          GestureDetector(
            onTap: () => adding.value = !adding.value,
            child: Icon(CupertinoIcons.plus, size: 16, color: c.textSecondary),
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
  final VoidCallback onDelete;
  final ValueChanged<bool> onToggle;

  const _GlossaryItem({
    required this.glossary,
    required this.selected,
    required this.onTap,
    required this.onDelete,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final name = glossary['name'] as String? ?? '';
    final enabled = glossary['enabled'] as bool? ?? true;
    final termCount = (glossary['term_count'] as num?)?.toInt() ?? 0;
    final c = AppColors.of(context);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        onSecondaryTap: onDelete,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          margin: const EdgeInsets.symmetric(vertical: 2),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: selected
                ? AppColors.blue.withValues(alpha: 0.2)
                : hovering.value
                ? c.cardBgHover
                : const Color(0x00000000),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              MacosSwitch(value: enabled, onChanged: onToggle),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      name,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: selected
                            ? FontWeight.w600
                            : FontWeight.w400,
                        color: selected ? AppColors.blue : c.textPrimary,
                      ),
                    ),
                    Text(
                      '$termCount 条词条',
                      style: TextStyle(fontSize: 11, color: c.textTertiary),
                    ),
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
    final addingTerm = useState(false);
    final name = glossary['name'] as String? ?? '';
    final enabled = glossary['enabled'] as bool? ?? true;
    final glossaryId = (glossary['id'] as num?)?.toInt() ?? 0;

    final c = AppColors.of(context);
    return Container(
      color: c.contentBg,
      child: Column(
        children: [
          _buildDetailHeader(context, name, enabled, glossaryId, c),
          _buildInfoCards(c),
          _buildSearchBar(searchController, addingTerm, c),
          if (addingTerm.value)
            _AddTermRow(
              onAdd: (t, tr, n) async {
                await notifier.addTerm(t, translation: tr, note: n);
                addingTerm.value = false;
              },
              onCancel: () => addingTerm.value = false,
            ),
          Expanded(
            child: loading
                ? const Center(child: ProgressCircle(radius: 12))
                : terms.isEmpty
                ? _buildEmptyState(glossaryId, c)
                : _buildTermList(context, glossaryId, c),
          ),
        ],
      ),
    );
  }

  Widget _buildDetailHeader(
    BuildContext context,
    String name,
    bool enabled,
    int glossaryId,
    AppColors c,
  ) {
    return Container(
      padding: const EdgeInsets.fromLTRB(24, 16, 24, 12),
      child: Row(
        children: [
          Text(
            name,
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w700,
              color: c.textPrimary,
              letterSpacing: -0.3,
            ),
          ),
          const SizedBox(width: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: enabled
                  ? AppColors.green.withValues(alpha: 0.15)
                  : c.textTertiary.withValues(alpha: 0.15),
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(
              enabled ? '已启用' : '已禁用',
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: enabled ? AppColors.green : c.textTertiary,
              ),
            ),
          ),
          const Spacer(),
          PushButton(
            controlSize: ControlSize.small,
            secondary: true,
            onPressed: () =>
                notifier.updateGlossary(glossaryId, enabled: !enabled),
            child: Text(
              enabled ? '禁用' : '启用',
              style: const TextStyle(fontSize: 13),
            ),
          ),
          MacosIconButton(
            icon: const Icon(
              CupertinoIcons.trash,
              size: 16,
              color: AppColors.red,
            ),
            onPressed: () => _confirmDelete(context, glossaryId),
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          ),
        ],
      ),
    );
  }

  Widget _buildInfoCards(AppColors c) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: AppColors.blue.withValues(alpha: 0.06),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: AppColors.blue.withValues(alpha: 0.12)),
        ),
        child: Row(
          children: [
            const Icon(
              CupertinoIcons.info_circle,
              size: 16,
              color: AppColors.blue,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                'AI 翻译自动使用已启用词库中的词条。多个启用词库时，按列表中最上的优先。',
                style: TextStyle(fontSize: 12, color: c.textSecondary),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildSearchBar(
    TextEditingController searchController,
    ValueNotifier<bool> addingTerm,
    AppColors c,
  ) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(24, 14, 24, 8),
      child: Row(
        children: [
          Expanded(
            child: MacosTextField(
              controller: searchController,
              placeholder: '搜索原文、译文或备注',
              prefix: Padding(
                padding: const EdgeInsets.only(left: 8),
                child: Icon(
                  CupertinoIcons.search,
                  size: 16,
                  color: c.textTertiary,
                ),
              ),
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
              style: TextStyle(fontSize: 13, color: c.textPrimary),
              onSubmitted: (q) => notifier.search(q),
              decoration: BoxDecoration(
                color: c.inputBg,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: c.borderLight),
              ),
            ),
          ),
          const SizedBox(width: 12),
          Text(
            '${terms.length} 条词条',
            style: TextStyle(fontSize: 12, color: c.textTertiary),
          ),
          const SizedBox(width: 12),
          PushButton(
            controlSize: ControlSize.regular,
            color: AppColors.blue,
            onPressed: () => addingTerm.value = !addingTerm.value,
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(CupertinoIcons.plus, size: 14, color: Color(0xFFFFFFFF)),
                SizedBox(width: 4),
                Text(
                  '新增词条',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFFFFFFFF),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyState(int glossaryId, AppColors c) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(CupertinoIcons.book, size: 48, color: c.textTertiary),
          const SizedBox(height: 12),
          Text(
            '这个词库还没有词条',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w600,
              color: c.textSecondary,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            '点击“新增词条”开始维护术语。',
            style: TextStyle(fontSize: 13, color: c.textTertiary),
          ),
        ],
      ),
    );
  }

  Widget _buildTermList(BuildContext context, int glossaryId, AppColors c) {
    return Column(
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
          decoration: BoxDecoration(
            color: c.surfaceBg,
            border: Border(
              bottom: BorderSide(color: c.borderLight, width: 0.5),
            ),
          ),
          child: Row(
            children: [
              Expanded(
                flex: 3,
                child: Text(
                  '原文',
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: c.textTertiary,
                  ),
                ),
              ),
              Expanded(
                flex: 3,
                child: Text(
                  '译文',
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: c.textTertiary,
                  ),
                ),
              ),
              Expanded(
                flex: 2,
                child: Text(
                  '备注',
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: c.textTertiary,
                  ),
                ),
              ),
              SizedBox(
                width: 40,
                child: Text(
                  '操作',
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: c.textTertiary,
                  ),
                  textAlign: TextAlign.right,
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: ListView.builder(
            padding: EdgeInsets.zero,
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
                onUpdate: (t, tr, n) =>
                    notifier.updateTerm(id, text: t, translation: tr, note: n),
              );
            },
          ),
        ),
      ],
    );
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
            onPressed: () => Navigator.pop(context),
          ),
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

/// 词条行（支持 hover 编辑/删除）
class _TermRow extends HookWidget {
  final String text;
  final String translation;
  final String note;
  final VoidCallback onDelete;
  final void Function(String text, String translation, String note) onUpdate;

  const _TermRow({
    required this.text,
    required this.translation,
    required this.note,
    required this.onDelete,
    required this.onUpdate,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final editing = useState(false);
    final textCtrl = useTextEditingController(text: text);
    final transCtrl = useTextEditingController(text: translation);
    final noteCtrl = useTextEditingController(text: note);
    final c = AppColors.of(context);

    void saveEdit() {
      if (textCtrl.text.trim().isEmpty) return;
      onUpdate(
        textCtrl.text.trim(),
        transCtrl.text.trim(),
        noteCtrl.text.trim(),
      );
      editing.value = false;
    }

    void cancelEdit() {
      textCtrl.text = text;
      transCtrl.text = translation;
      noteCtrl.text = note;
      editing.value = false;
    }

    if (editing.value) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
        decoration: BoxDecoration(
          color: c.surfaceBg,
          border: Border(bottom: BorderSide(color: c.borderLight, width: 0.5)),
        ),
        child: Row(
          children: [
            Expanded(
              flex: 3,
              child: MacosTextField(
                controller: textCtrl,
                style: TextStyle(fontSize: 13, color: c.textPrimary),
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                decoration: BoxDecoration(
                  color: c.inputBg,
                  borderRadius: BorderRadius.circular(4),
                ),
                placeholder: '原文',
                placeholderStyle: TextStyle(
                  fontSize: 13,
                  color: c.textPlaceholder,
                ),
                onSubmitted: (_) => saveEdit(),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              flex: 3,
              child: MacosTextField(
                controller: transCtrl,
                style: const TextStyle(fontSize: 13, color: AppColors.blue),
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                decoration: BoxDecoration(
                  color: c.inputBg,
                  borderRadius: BorderRadius.circular(4),
                ),
                placeholder: '译文',
                placeholderStyle: TextStyle(
                  fontSize: 13,
                  color: c.textPlaceholder,
                ),
                onSubmitted: (_) => saveEdit(),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              flex: 2,
              child: MacosTextField(
                controller: noteCtrl,
                style: TextStyle(fontSize: 11, color: c.textTertiary),
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                decoration: BoxDecoration(
                  color: c.inputBg,
                  borderRadius: BorderRadius.circular(4),
                ),
                placeholder: '备注',
                placeholderStyle: TextStyle(
                  fontSize: 11,
                  color: c.textPlaceholder,
                ),
                onSubmitted: (_) => saveEdit(),
              ),
            ),
            const SizedBox(width: 8),
            GestureDetector(
              onTap: saveEdit,
              child: const Icon(
                CupertinoIcons.checkmark_circle_fill,
                size: 16,
                color: AppColors.green,
              ),
            ),
            const SizedBox(width: 6),
            GestureDetector(
              onTap: cancelEdit,
              child: Icon(
                CupertinoIcons.xmark_circle_fill,
                size: 16,
                color: c.textTertiary,
              ),
            ),
          ],
        ),
      );
    }

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onDoubleTap: () => editing.value = true,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
          decoration: BoxDecoration(
            color: hovering.value ? c.cardBgHover : const Color(0x00000000),
            border: Border(
              bottom: BorderSide(color: c.borderLight, width: 0.5),
            ),
          ),
          child: Row(
            children: [
              Expanded(
                flex: 3,
                child: Text(
                  text,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                    color: c.textPrimary,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Expanded(
                flex: 3,
                child: Text(
                  translation,
                  style: const TextStyle(fontSize: 13, color: AppColors.blue),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              Expanded(
                flex: 2,
                child: Text(
                  note.isEmpty ? '—' : note,
                  style: TextStyle(fontSize: 11, color: c.textTertiary),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              SizedBox(
                width: 60,
                child: hovering.value
                    ? Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          GestureDetector(
                            onTap: () => editing.value = true,
                            child: Icon(
                              CupertinoIcons.pencil,
                              size: 14,
                              color: c.textTertiary,
                            ),
                          ),
                          const SizedBox(width: 8),
                          GestureDetector(
                            onTap: onDelete,
                            child: const Icon(
                              CupertinoIcons.trash,
                              size: 14,
                              color: AppColors.red,
                            ),
                          ),
                        ],
                      )
                    : const SizedBox.shrink(),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 行内新增词条输入行
class _AddTermRow extends HookWidget {
  final Future<void> Function(String text, String translation, String note)
  onAdd;
  final VoidCallback onCancel;

  const _AddTermRow({required this.onAdd, required this.onCancel});

  @override
  Widget build(BuildContext context) {
    final textCtrl = useTextEditingController();
    final transCtrl = useTextEditingController();
    final noteCtrl = useTextEditingController();
    final c = AppColors.of(context);

    void submit() {
      if (textCtrl.text.trim().isEmpty) return;
      onAdd(textCtrl.text.trim(), transCtrl.text.trim(), noteCtrl.text.trim());
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
      decoration: BoxDecoration(
        color: c.surfaceBg,
        border: Border(bottom: BorderSide(color: c.borderLight, width: 0.5)),
      ),
      child: Row(
        children: [
          Expanded(
            flex: 3,
            child: MacosTextField(
              controller: textCtrl,
              autofocus: true,
              style: TextStyle(fontSize: 13, color: c.textPrimary),
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
              decoration: BoxDecoration(
                color: c.inputBg,
                borderRadius: BorderRadius.circular(4),
              ),
              placeholder: '原文（必填）',
              placeholderStyle: TextStyle(
                fontSize: 13,
                color: c.textPlaceholder,
              ),
              onSubmitted: (_) => submit(),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            flex: 3,
            child: MacosTextField(
              controller: transCtrl,
              style: const TextStyle(fontSize: 13, color: AppColors.blue),
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
              decoration: BoxDecoration(
                color: c.inputBg,
                borderRadius: BorderRadius.circular(4),
              ),
              placeholder: '译文',
              placeholderStyle: TextStyle(
                fontSize: 13,
                color: c.textPlaceholder,
              ),
              onSubmitted: (_) => submit(),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            flex: 2,
            child: MacosTextField(
              controller: noteCtrl,
              style: TextStyle(fontSize: 11, color: c.textTertiary),
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
              decoration: BoxDecoration(
                color: c.inputBg,
                borderRadius: BorderRadius.circular(4),
              ),
              placeholder: '备注',
              placeholderStyle: TextStyle(
                fontSize: 11,
                color: c.textPlaceholder,
              ),
              onSubmitted: (_) => submit(),
            ),
          ),
          const SizedBox(width: 8),
          GestureDetector(
            onTap: submit,
            child: const Icon(
              CupertinoIcons.checkmark_circle_fill,
              size: 16,
              color: AppColors.green,
            ),
          ),
          const SizedBox(width: 6),
          GestureDetector(
            onTap: onCancel,
            child: Icon(
              CupertinoIcons.xmark_circle_fill,
              size: 16,
              color: c.textTertiary,
            ),
          ),
        ],
      ),
    );
  }
}

/// 小型操作按钮
class _SmallActionBtn extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;
  const _SmallActionBtn({
    required this.icon,
    required this.label,
    required this.onTap,
  });
  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        decoration: BoxDecoration(
          color: c.inputBg,
          borderRadius: BorderRadius.circular(6),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 13, color: c.textTertiary),
            const SizedBox(width: 4),
            Text(label, style: TextStyle(fontSize: 12, color: c.textPrimary)),
          ],
        ),
      ),
    );
  }
}
