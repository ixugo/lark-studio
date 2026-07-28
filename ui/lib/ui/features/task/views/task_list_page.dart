import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart'
    show
        CupertinoAlertDialog,
        CupertinoColors,
        CupertinoDialogAction,
        CupertinoIcons,
        CupertinoPageRoute,
        CupertinoSlidingSegmentedControl,
        showCupertinoDialog;
import 'package:flutter/widgets.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/task.dart';
import '../../../../data/services/api_client.dart';
import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../../core/glass_card.dart';
import '../view_models/task_list_view_model.dart';
import 'task_detail_page.dart';

class TaskListPage extends HookConsumerWidget {
  const TaskListPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ts = ref.watch(taskListProvider);
    final notifier = ref.read(taskListProvider.notifier);
    final ws = ref.read(wsServiceProvider);

    useEffect(() {
      Future.microtask(() => notifier.refresh());
      final sub = ws.events.listen((event) {
        if (event.type.startsWith('task_')) {
          notifier.refresh();
        }
      });
      return sub.cancel;
    }, const []);

    return _buildBody(context, ts, notifier);
  }

  Widget _buildBody(
    BuildContext context,
    TaskListState ts,
    TaskListNotifier notifier,
  ) {
    final c = AppColors.of(context);
    if (ts.loading && ts.tasks.isEmpty) {
      return const Center(child: ProgressCircle(radius: 14));
    }
    if (ts.error != null && ts.tasks.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              CupertinoIcons.exclamationmark_circle,
              size: 48,
              color: c.textQuaternary,
            ),
            const SizedBox(height: 12),
            Text(
              ts.error!,
              style: TextStyle(color: c.textTertiary, fontSize: 14),
            ),
            const SizedBox(height: 16),
            AppButton.secondary(
              onPressed: () => notifier.refresh(),
              child: const Text('重试'),
            ),
          ],
        ),
      );
    }
    if (ts.tasks.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(CupertinoIcons.film, size: 56, color: c.textQuaternary),
            const SizedBox(height: 12),
            Text('暂无任务', style: TextStyle(color: c.textTertiary, fontSize: 15)),
            const SizedBox(height: 4),
            Text(
              '请从启动台选择任务链开始',
              style: TextStyle(color: c.textQuaternary, fontSize: 13),
            ),
          ],
        ),
      );
    }

    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(28, 20, 28, 28),
      itemCount: ts.tasks.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final t = ts.tasks[index];
        return _TaskCard(
          task: t,
          onDelete: () => notifier.deleteTask(t.id),
          onPause: t.canPause ? () => notifier.pauseTask(t.id) : null,
          onResume: t.canResume ? () => notifier.resumeTask(t.id) : null,
        );
      },
    );
  }
}

class _TaskCard extends HookWidget {
  final Task task;
  final VoidCallback onDelete;
  final VoidCallback? onPause;
  final VoidCallback? onResume;
  const _TaskCard({
    required this.task,
    required this.onDelete,
    this.onPause,
    this.onResume,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final confirmingDelete = useState(false);
    return GlassCard(
      radius: 14,
      onTap: () {
        Navigator.of(context).push(
          CupertinoPageRoute(builder: (_) => TaskDetailPage(taskId: task.id)),
        );
      },
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            _statusIcon(task.status, c),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    task.fileName,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      color: c.textPrimary,
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 4),
                  Row(
                    children: [
                      _badge(task.modeName, c.accent),
                      const SizedBox(width: 8),
                      _badge(task.statusName, _statusColor(task.status)),
                      if (task.currentStep.isNotEmpty) ...[
                        const SizedBox(width: 8),
                        Text(
                          task.currentStep,
                          style: TextStyle(fontSize: 12, color: c.textTertiary),
                        ),
                      ],
                    ],
                  ),
                ],
              ),
            ),
            if (onPause != null)
              MacosIconButton(
                icon: const Icon(
                  CupertinoIcons.pause_circle,
                  size: 20,
                  color: CupertinoColors.systemOrange,
                ),
                onPressed: onPause,
                padding: EdgeInsets.zero,
              ),
            if (onResume != null)
              MacosIconButton(
                icon: const Icon(
                  CupertinoIcons.play_circle,
                  size: 20,
                  color: CupertinoColors.systemGreen,
                ),
                onPressed: onResume,
                padding: EdgeInsets.zero,
              ),
            if (confirmingDelete.value) ...[
              Text(
                '确认删除？',
                style: TextStyle(fontSize: 11, color: c.textSecondary),
              ),
              const SizedBox(width: 6),
              AppButton.secondary(
                onPressed: () => confirmingDelete.value = false,
                child: const Text('取消'),
              ),
              const SizedBox(width: 4),
              AppButton(
                color: AppColors.red,
                onPressed: onDelete,
                child: const Text('删除'),
              ),
            ] else
              MacosIconButton(
                icon: Icon(
                  CupertinoIcons.trash,
                  size: 16,
                  color: c.textQuaternary,
                ),
                onPressed: () => confirmingDelete.value = true,
                padding: EdgeInsets.zero,
              ),
            const SizedBox(width: 4),
            Icon(
              CupertinoIcons.chevron_right,
              size: 14,
              color: c.textQuaternary,
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusIcon(int status, AppColors c) {
    switch (status) {
      case 1:
        return const ProgressCircle(radius: 10);
      case 2:
        return const Icon(
          CupertinoIcons.pause_circle_fill,
          color: CupertinoColors.systemYellow,
          size: 22,
        );
      case 3:
        return const Icon(
          CupertinoIcons.checkmark_circle_fill,
          color: CupertinoColors.systemGreen,
          size: 22,
        );
      case 4:
        return const Icon(
          CupertinoIcons.xmark_circle_fill,
          color: CupertinoColors.systemRed,
          size: 22,
        );
      default:
        return Icon(CupertinoIcons.clock, color: c.textQuaternary, size: 22);
    }
  }

  Color _statusColor(int status) {
    switch (status) {
      case 1:
        return CupertinoColors.systemOrange;
      case 2:
        return CupertinoColors.systemYellow;
      case 3:
        return CupertinoColors.systemGreen;
      case 4:
        return CupertinoColors.systemRed;
      default:
        return const Color(0xFF8E8E93);
    }
  }

  Widget _badge(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 11,
          color: color,
          fontWeight: FontWeight.w500,
        ),
      ),
    );
  }
}

// ignore: unused_element
class _BatchImportSheet extends HookConsumerWidget {
  const _BatchImportSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final c = AppColors.of(context);
    final langController = useTextEditingController(text: 'zh');
    final selectedFiles = useState(<String>[]);
    final mode = useState(3);
    final submitting = useState(false);

    Future<void> pickFiles() async {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.video,
        allowMultiple: true,
        dialogTitle: '选择视频文件',
      );
      if (result != null) {
        final updated = [...selectedFiles.value];
        for (final f in result.files) {
          if (f.path != null && !updated.contains(f.path)) updated.add(f.path!);
        }
        selectedFiles.value = updated;
      }
    }

    Future<void> submit() async {
      submitting.value = true;
      try {
        final count = await ref
            .read(taskListProvider.notifier)
            .batchCreateTasks(
              videos: List.from(selectedFiles.value),
              mode: mode.value,
              targetLang: langController.text.trim(),
            );
        if (context.mounted) {
          Navigator.pop(context);
          _showInfo(context, '已创建 $count 个任务');
        }
      } on ApiException catch (e) {
        if (context.mounted) _showInfo(context, e.message);
      } catch (e) {
        if (context.mounted) _showInfo(context, '批量创建失败: $e');
      }
      submitting.value = false;
    }

    return Container(
      height: MediaQuery.of(context).size.height * 0.65,
      decoration: BoxDecoration(
        color: c.contentBg,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(color: c.borderLight, width: 0.5),
              ),
            ),
            child: Row(
              children: [
                GestureDetector(
                  onTap: () => Navigator.pop(context),
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Text(
                      '取消',
                      style: TextStyle(fontSize: 15, color: c.accent),
                    ),
                  ),
                ),
                Expanded(
                  child: Text(
                    '批量导入',
                    textAlign: TextAlign.center,
                    style: TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w600,
                      color: c.textPrimary,
                    ),
                  ),
                ),
                GestureDetector(
                  onTap: (submitting.value || selectedFiles.value.isEmpty)
                      ? null
                      : submit,
                  child: MouseRegion(
                    cursor: SystemMouseCursors.click,
                    child: Text(
                      '导入 (${selectedFiles.value.length})',
                      style: TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                        color: (submitting.value || selectedFiles.value.isEmpty)
                            ? c.textQuaternary
                            : c.accent,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.all(20),
              children: [
                Row(
                  children: [
                    Text(
                      '已选视频',
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                        color: c.textTertiary,
                      ),
                    ),
                    const Spacer(),
                    AppButton(
                      onPressed: pickFiles,
                      child: const Text('选择视频文件'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (selectedFiles.value.isEmpty)
                  Container(
                    padding: const EdgeInsets.all(24),
                    decoration: _boxDecoration(c),
                    child: Center(
                      child: Text(
                        '点击右上角选择视频文件',
                        style: TextStyle(fontSize: 13, color: c.textQuaternary),
                      ),
                    ),
                  )
                else
                  Container(
                    constraints: const BoxConstraints(maxHeight: 150),
                    padding: const EdgeInsets.all(12),
                    decoration: _boxDecoration(c),
                    child: ListView.builder(
                      shrinkWrap: true,
                      itemCount: selectedFiles.value.length,
                      itemBuilder: (_, i) {
                        final name = selectedFiles.value[i].split('/').last;
                        return Padding(
                          padding: const EdgeInsets.symmetric(vertical: 3),
                          child: Row(
                            children: [
                              Icon(
                                CupertinoIcons.film,
                                size: 14,
                                color: c.textTertiary,
                              ),
                              const SizedBox(width: 6),
                              Expanded(
                                child: Text(
                                  name,
                                  style: TextStyle(
                                    fontSize: 12,
                                    color: c.textSecondary,
                                  ),
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                              GestureDetector(
                                onTap: () {
                                  final u = [...selectedFiles.value];
                                  u.removeAt(i);
                                  selectedFiles.value = u;
                                },
                                child: Icon(
                                  CupertinoIcons.xmark_circle_fill,
                                  size: 16,
                                  color: c.textQuaternary,
                                ),
                              ),
                            ],
                          ),
                        );
                      },
                    ),
                  ),
                const SizedBox(height: 20),
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: Text(
                    '处理模式',
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w500,
                      color: c.textTertiary,
                    ),
                  ),
                ),
                CupertinoSlidingSegmentedControl<int>(
                  groupValue: mode.value,
                  children: const {
                    1: Padding(
                      padding: EdgeInsets.symmetric(horizontal: 12),
                      child: Text('字幕', style: TextStyle(fontSize: 13)),
                    ),
                    2: Padding(
                      padding: EdgeInsets.symmetric(horizontal: 12),
                      child: Text('翻译', style: TextStyle(fontSize: 13)),
                    ),
                    3: Padding(
                      padding: EdgeInsets.symmetric(horizontal: 12),
                      child: Text('配音', style: TextStyle(fontSize: 13)),
                    ),
                  },
                  onValueChanged: (v) => mode.value = v!,
                ),
                const SizedBox(height: 20),
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: Text(
                    '目标语言',
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w500,
                      color: c.textTertiary,
                    ),
                  ),
                ),
                MacosTextField(
                  controller: langController,
                  placeholder: 'zh (默认中文)',
                  padding: const EdgeInsets.all(12),
                  decoration: _boxDecoration(c),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  BoxDecoration _boxDecoration(AppColors c) => BoxDecoration(
    color: c.inputBg,
    borderRadius: BorderRadius.circular(8),
    border: Border.all(color: c.borderLight, width: 0.5),
  );

  void _showInfo(BuildContext context, String msg) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
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
