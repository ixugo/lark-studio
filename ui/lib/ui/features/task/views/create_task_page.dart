import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart'
    show
        CupertinoAlertDialog,
        CupertinoDialogAction,
        CupertinoIcons,
        showCupertinoDialog;
import 'package:flutter/widgets.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/services/api_client.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../../core/glass_card.dart';
import '../view_models/task_list_view_model.dart';

/// 新建任务向导：居中玻璃浮层 = 选文件 → 选流程 → 配置 → 创建
class CreateTaskPage extends HookConsumerWidget {
  final int initialMode;

  const CreateTaskPage({super.key, this.initialMode = 3});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final c = AppColors.of(context);
    final pathController = useTextEditingController();
    final outputController = useTextEditingController();
    final langController = useTextEditingController(text: 'zh');
    final mode = useState(initialMode);
    final submitting = useState(false);
    // 选中文件后刷新 dropzone 文案
    useListenable(pathController);

    Future<void> pickFile() async {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.video,
        dialogTitle: '选择视频文件',
      );
      if (result != null && result.files.single.path != null) {
        pathController.text = result.files.single.path!;
      }
    }

    Future<void> submit() async {
      if (pathController.text.trim().isEmpty) return;
      submitting.value = true;
      try {
        await ref
            .read(taskListProvider.notifier)
            .createTask(
              inputPath: pathController.text.trim(),
              mode: mode.value,
              outputDir: outputController.text.trim(),
              targetLang: langController.text.trim(),
            );
        if (context.mounted) Navigator.pop(context);
      } on ApiException catch (e) {
        if (context.mounted) _showError(context, e.message);
      } catch (e) {
        if (context.mounted) _showError(context, '创建失败: $e');
      }
      submitting.value = false;
    }

    final canSubmit =
        pathController.text.trim().isNotEmpty && !submitting.value;

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 620),
        child: SizedBox(
          width: MediaQuery.sizeOf(context).width - 80,
          child: GlassCard(
            radius: 18,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                _SheetHeader(c: c),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 16, 20, 8),
                  child: Column(
                    children: [
                      _DropZone(
                        c: c,
                        filePath: pathController.text,
                        onTap: pickFile,
                      ),
                      const SizedBox(height: 16),
                      _ModePills(mode: mode, c: c),
                      const SizedBox(height: 16),
                      Row(
                        children: [
                          Expanded(
                            child: _LabeledField(
                              label: '目标语言',
                              c: c,
                              child: MacosTextField(
                                controller: langController,
                                placeholder: 'zh (默认中文)',
                                padding: const EdgeInsets.all(10),
                                decoration: _fieldDecoration(c),
                              ),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: _LabeledField(
                              label: '输出目录（留空自动生成）',
                              c: c,
                              child: MacosTextField(
                                controller: outputController,
                                placeholder: '视频同级目录',
                                padding: const EdgeInsets.all(10),
                                decoration: _fieldDecoration(c),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                _SheetFooter(c: c, canSubmit: canSubmit, onSubmit: submit),
              ],
            ),
          ),
        ),
      ),
    );
  }

  BoxDecoration _fieldDecoration(AppColors c) => BoxDecoration(
    color: c.inputBg,
    borderRadius: BorderRadius.circular(8),
    border: Border.all(color: c.borderLight, width: 0.5),
  );

  void _showError(BuildContext context, String msg) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('错误'),
        content: Text(msg),
        actions: [
          CupertinoDialogAction(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }
}

/// 浮层头部：标题 + 关闭按钮
class _SheetHeader extends StatelessWidget {
  final AppColors c;
  const _SheetHeader({required this.c});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(20, 14, 12, 12),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: c.glassStroke, width: 0.5)),
      ),
      child: Row(
        children: [
          Text(
            '新建任务',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w700,
              color: c.textPrimary,
            ),
          ),
          const SizedBox(width: 20),
          const Expanded(child: _WizardSteps()),
          GestureDetector(
            onTap: () => Navigator.pop(context),
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Icon(
                CupertinoIcons.xmark,
                size: 16,
                color: c.textTertiary,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// 文件选择区：点击打开文件选择器，选中后显示文件名
class _DropZone extends HookWidget {
  final AppColors c;
  final String filePath;
  final VoidCallback onTap;
  const _DropZone({
    required this.c,
    required this.filePath,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final hasFile = filePath.trim().isNotEmpty;
    final fileName = hasFile ? filePath.split('/').last : '';

    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 140),
          width: double.infinity,
          padding: const EdgeInsets.symmetric(vertical: 30),
          decoration: BoxDecoration(
            color: hovering.value || hasFile
                ? c.accent.withValues(alpha: 0.08)
                : c.glassCardBg,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: hasFile ? c.accent.withValues(alpha: 0.45) : c.glassStroke,
              width: 1,
            ),
          ),
          child: Column(
            children: [
              Icon(
                hasFile
                    ? CupertinoIcons.checkmark_circle_fill
                    : CupertinoIcons.arrow_up_doc,
                size: 28,
                color: hasFile ? c.accent : c.textTertiary,
              ),
              const SizedBox(height: 8),
              Text(
                hasFile ? fileName : '点击选择视频文件',
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: c.textPrimary,
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              const SizedBox(height: 4),
              Text(
                hasFile ? '再次点击可更换' : '支持 MP4 / MKV / MOV / MP3 / WAV',
                style: TextStyle(fontSize: 11.5, color: c.textTertiary),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 处理模式三选一：字幕 / 翻译 / 配音
class _ModePills extends StatelessWidget {
  final ValueNotifier<int> mode;
  final AppColors c;
  const _ModePills({required this.mode, required this.c});

  static const _modes = [
    (1, '字幕', '听写 → 字幕'),
    (2, '翻译', '听写 → 翻译 → 字幕'),
    (3, '配音', '听写 → 翻译 → 配音 → 合成'),
  ];

  @override
  Widget build(BuildContext context) {
    return Row(
      children: _modes.map((m) {
        final (value, title, desc) = m;
        final selected = mode.value == value;
        return Expanded(
          child: GestureDetector(
            onTap: () => mode.value = value,
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 140),
                margin: EdgeInsets.only(right: value != 3 ? 10 : 0),
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 10,
                ),
                decoration: BoxDecoration(
                  color: selected
                      ? c.accent.withValues(alpha: 0.12)
                      : c.glassCardBg,
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(
                    color: selected
                        ? c.accent.withValues(alpha: 0.5)
                        : c.glassStroke,
                    width: 0.5,
                  ),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                        color: selected ? c.accent : c.textPrimary,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      desc,
                      style: TextStyle(fontSize: 10.5, color: c.textTertiary),
                    ),
                  ],
                ),
              ),
            ),
          ),
        );
      }).toList(),
    );
  }
}

class _LabeledField extends StatelessWidget {
  final String label;
  final Widget child;
  final AppColors c;
  const _LabeledField({
    required this.label,
    required this.child,
    required this.c,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(bottom: 6),
          child: Text(
            label,
            style: TextStyle(fontSize: 12, color: c.textTertiary),
          ),
        ),
        child,
      ],
    );
  }
}

/// 向导步骤说明用于显示创建流程，不改变提交行为。
class _WizardSteps extends StatelessWidget {
  const _WizardSteps();

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    const steps = ['选文件', '选流程', '开始'];
    return Row(
      mainAxisAlignment: MainAxisAlignment.end,
      children: steps.asMap().entries.map((entry) {
        final active = entry.key == 0;
        return Padding(
          padding: const EdgeInsets.only(left: 10),
          child: Text(
            '${entry.key + 1} ${entry.value}',
            style: TextStyle(
              fontSize: 11,
              fontWeight: active ? FontWeight.w600 : FontWeight.w400,
              color: active ? c.accent : c.textTertiary,
            ),
          ),
        );
      }).toList(),
    );
  }
}

/// 浮层底部：取消 / 创建
class _SheetFooter extends StatelessWidget {
  final AppColors c;
  final bool canSubmit;
  final Future<void> Function() onSubmit;
  const _SheetFooter({
    required this.c,
    required this.canSubmit,
    required this.onSubmit,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(20, 12, 20, 14),
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: c.glassStroke, width: 0.5)),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          AppButton.secondary(
            onPressed: () => Navigator.pop(context),
            child: const Text('取消'),
          ),
          const SizedBox(width: 10),
          AppButton(
            onPressed: canSubmit ? () => onSubmit() : null,
            child: const Text(
              '开始任务',
              style: TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ),
    );
  }
}
