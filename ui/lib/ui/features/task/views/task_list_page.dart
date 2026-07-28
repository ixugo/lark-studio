import 'dart:async';
import 'dart:ui';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart' show CupertinoAlertDialog, CupertinoColors, CupertinoDialogAction, CupertinoIcons, CupertinoPageRoute, CupertinoSlidingSegmentedControl, showCupertinoDialog, showCupertinoModalPopup;
import 'package:flutter/widgets.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/task.dart';
import '../../../../data/services/api_client.dart';
import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../view_models/task_list_view_model.dart';
import 'create_task_page.dart';
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

    return Column(
      children: [
        _buildToolbar(context, notifier),
        Expanded(child: _buildBody(context, ts, notifier)),
      ],
    );
  }

  Widget _buildToolbar(BuildContext context, TaskListNotifier notifier) {
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          height: 52,
          padding: const EdgeInsets.symmetric(horizontal: 28),
          decoration: BoxDecoration(
            color: const Color(0xFFFFFFFF).withValues(alpha: 0.65),
            border: const Border(
                bottom: BorderSide(color: Color(0x1A000000), width: 0.5)),
          ),
          child: Row(
            children: [
              const Text('任务',
                  style: TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.w700,
                      color: Color(0xFF1D1D1F),
                      letterSpacing: -0.5)),
              const Spacer(),
              MacosIconButton(
                icon: const Icon(CupertinoIcons.refresh,
                    size: 16, color: Color(0xFF8E8E93)),
                onPressed: () => notifier.refresh(),
              ),
              const SizedBox(width: 10),
              AppButton.secondary(
                onPressed: () => _showBatchDialog(context),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(CupertinoIcons.folder_badge_plus, size: 15),
                    SizedBox(width: 5),
                    Text('批量导入'),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              AppButton(
                onPressed: () => _showCreateDialog(context),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(CupertinoIcons.add, size: 15),
                    SizedBox(width: 5),
                    Text('新建', style: TextStyle(fontWeight: FontWeight.w600)),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildBody(
      BuildContext context, TaskListState ts, TaskListNotifier notifier) {
    if (ts.loading && ts.tasks.isEmpty) {
      return const Center(child: ProgressCircle(radius: 14));
    }
    if (ts.error != null && ts.tasks.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(CupertinoIcons.exclamationmark_circle,
                size: 48, color: Color(0xFFC7C7CC)),
            const SizedBox(height: 12),
            Text(ts.error!,
                style:
                    const TextStyle(color: Color(0xFF8E8E93), fontSize: 14)),
            const SizedBox(height: 16),
            AppButton.secondary(
                onPressed: () => notifier.refresh(),
                child: const Text('重试')),
          ],
        ),
      );
    }
    if (ts.tasks.isEmpty) {
      return const Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(CupertinoIcons.film, size: 56, color: Color(0xFFC7C7CC)),
            SizedBox(height: 12),
            Text('暂无任务',
                style: TextStyle(color: Color(0xFF8E8E93), fontSize: 15)),
            SizedBox(height: 4),
            Text('点击右上角新建任务开始使用',
                style: TextStyle(color: Color(0xFFC7C7CC), fontSize: 13)),
          ],
        ),
      );
    }

    return ListView.separated(
      padding: const EdgeInsets.fromLTRB(28, 20, 28, 28),
      itemCount: ts.tasks.length,
      separatorBuilder: (_, __) => const SizedBox(height: 10),
      itemBuilder: (context, index) {
        final t = ts.tasks[index];
        return _TaskCard(
          task: t,
          onDelete: () => _confirmDelete(context, t, notifier),
          onPause: t.canPause ? () => notifier.pauseTask(t.id) : null,
          onResume: t.canResume ? () => notifier.resumeTask(t.id) : null,
        );
      },
    );
  }

  void _showCreateDialog(BuildContext context) {
    showCupertinoModalPopup(
        context: context, builder: (_) => const CreateTaskPage());
  }

  void _showBatchDialog(BuildContext context) {
    showCupertinoModalPopup(
        context: context, builder: (_) => const _BatchImportSheet());
  }

  void _confirmDelete(
      BuildContext context, Task task, TaskListNotifier notifier) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('删除任务'),
        content: Text('确定删除 "${task.fileName}" 吗？'),
        actions: [
          CupertinoDialogAction(
              isDefaultAction: true,
              child: const Text('取消'),
              onPressed: () => Navigator.pop(context)),
          CupertinoDialogAction(
            isDestructiveAction: true,
            child: const Text('删除'),
            onPressed: () {
              Navigator.pop(context);
              notifier.deleteTask(task.id);
            },
          ),
        ],
      ),
    );
  }
}

class _TaskCard extends StatelessWidget {
  final Task task;
  final VoidCallback onDelete;
  final VoidCallback? onPause;
  final VoidCallback? onResume;
  const _TaskCard(
      {required this.task,
      required this.onDelete,
      this.onPause,
      this.onResume});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () {
        Navigator.of(context).push(
            CupertinoPageRoute(builder: (_) => TaskDetailPage(taskId: task.id)));
      },
      child: ClipRRect(
        borderRadius: BorderRadius.circular(14),
        child: BackdropFilter(
          filter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
          child: Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [
                  const Color(0xFFFFFFFF).withValues(alpha: 0.80),
                  const Color(0xFFF9F9FB).withValues(alpha: 0.70),
                ],
              ),
              borderRadius: BorderRadius.circular(14),
              border: Border.all(
                  color: const Color(0xFFFFFFFF).withValues(alpha: 0.5),
                  width: 0.5),
              boxShadow: [
                BoxShadow(
                    color: const Color(0xFF000000).withValues(alpha: 0.04),
                    blurRadius: 14,
                    offset: const Offset(0, 3)),
              ],
            ),
            child: Row(
              children: [
                _statusIcon(task.status),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(task.fileName,
                          style: const TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w500,
                              color: Color(0xFF1D1D1F)),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis),
                      const SizedBox(height: 4),
                      Row(
                        children: [
                          _badge(task.modeName, CupertinoColors.systemBlue),
                          const SizedBox(width: 8),
                          _badge(task.statusName, _statusColor(task.status)),
                          if (task.currentStep.isNotEmpty) ...[
                            const SizedBox(width: 8),
                            Text(task.currentStep,
                                style: const TextStyle(
                                    fontSize: 12, color: Color(0xFF8E8E93))),
                          ],
                        ],
                      ),
                    ],
                  ),
                ),
                if (onPause != null)
                  MacosIconButton(
                    icon: const Icon(CupertinoIcons.pause_circle,
                        size: 20, color: CupertinoColors.systemOrange),
                    onPressed: onPause,
                    padding: EdgeInsets.zero,
                  ),
                if (onResume != null)
                  MacosIconButton(
                    icon: const Icon(CupertinoIcons.play_circle,
                        size: 20, color: CupertinoColors.systemGreen),
                    onPressed: onResume,
                    padding: EdgeInsets.zero,
                  ),
                MacosIconButton(
                  icon: const Icon(CupertinoIcons.trash,
                      size: 16, color: Color(0xFFC7C7CC)),
                  onPressed: onDelete,
                  padding: EdgeInsets.zero,
                ),
                const SizedBox(width: 4),
                const Icon(CupertinoIcons.chevron_right,
                    size: 14, color: Color(0xFFC7C7CC)),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _statusIcon(int status) {
    switch (status) {
      case 1: return const ProgressCircle(radius: 10);
      case 2: return const Icon(CupertinoIcons.pause_circle_fill, color: CupertinoColors.systemYellow, size: 22);
      case 3: return const Icon(CupertinoIcons.checkmark_circle_fill, color: CupertinoColors.systemGreen, size: 22);
      case 4: return const Icon(CupertinoIcons.xmark_circle_fill, color: CupertinoColors.systemRed, size: 22);
      default: return const Icon(CupertinoIcons.clock, color: Color(0xFFC7C7CC), size: 22);
    }
  }

  Color _statusColor(int status) {
    switch (status) {
      case 1: return CupertinoColors.systemOrange;
      case 2: return CupertinoColors.systemYellow;
      case 3: return CupertinoColors.systemGreen;
      case 4: return CupertinoColors.systemRed;
      default: return const Color(0xFF8E8E93);
    }
  }

  Widget _badge(String text, Color color) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
          color: color.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(4)),
      child: Text(text,
          style: TextStyle(
              fontSize: 11, color: color, fontWeight: FontWeight.w500)),
    );
  }
}

class _BatchImportSheet extends HookConsumerWidget {
  const _BatchImportSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final langController = useTextEditingController(text: 'zh');
    final selectedFiles = useState(<String>[]);
    final mode = useState(3);
    final submitting = useState(false);

    Future<void> pickFiles() async {
      final result = await FilePicker.platform.pickFiles(
          type: FileType.video, allowMultiple: true, dialogTitle: '选择视频文件');
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
        final count = await ref.read(taskListProvider.notifier).batchCreateTasks(
            videos: List.from(selectedFiles.value),
            mode: mode.value,
            targetLang: langController.text.trim());
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
      decoration: const BoxDecoration(
        color: CupertinoColors.systemGroupedBackground,
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
            decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5))),
            child: Row(children: [
              GestureDetector(onTap: () => Navigator.pop(context), child: const MouseRegion(cursor: SystemMouseCursors.click, child: Text('取消', style: TextStyle(fontSize: 15, color: Color(0xFF007AFF))))),
              const Expanded(child: Text('批量导入', textAlign: TextAlign.center, style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F)))),
              GestureDetector(onTap: (submitting.value || selectedFiles.value.isEmpty) ? null : submit,
                  child: MouseRegion(cursor: SystemMouseCursors.click, child: Text('导入 (${selectedFiles.value.length})', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: (submitting.value || selectedFiles.value.isEmpty) ? const Color(0xFFC7C7CC) : const Color(0xFF007AFF))))),
            ]),
          ),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.all(20),
              children: [
                Row(children: [
                  const Text('已选视频', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93))),
                  const Spacer(),
                  AppButton(onPressed: pickFiles, child: const Text('选择视频文件')),
                ]),
                const SizedBox(height: 8),
                if (selectedFiles.value.isEmpty)
                  Container(padding: const EdgeInsets.all(24), decoration: BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8), border: Border.all(color: const Color(0xFFE5E5EA))), child: const Center(child: Text('点击右上角选择视频文件', style: TextStyle(fontSize: 13, color: Color(0xFFC7C7CC)))))
                else
                  Container(
                    constraints: const BoxConstraints(maxHeight: 150),
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8), border: Border.all(color: const Color(0xFFE5E5EA))),
                    child: ListView.builder(
                      shrinkWrap: true, itemCount: selectedFiles.value.length,
                      itemBuilder: (_, i) {
                        final name = selectedFiles.value[i].split('/').last;
                        return Padding(padding: const EdgeInsets.symmetric(vertical: 3), child: Row(children: [
                          const Icon(CupertinoIcons.film, size: 14, color: Color(0xFF8E8E93)),
                          const SizedBox(width: 6),
                          Expanded(child: Text(name, style: const TextStyle(fontSize: 12, color: Color(0xFF3A3A3C)), maxLines: 1, overflow: TextOverflow.ellipsis)),
                          GestureDetector(onTap: () { final u = [...selectedFiles.value]; u.removeAt(i); selectedFiles.value = u; }, child: const Icon(CupertinoIcons.xmark_circle_fill, size: 16, color: Color(0xFFC7C7CC))),
                        ]));
                      },
                    ),
                  ),
                const SizedBox(height: 20),
                const Padding(padding: EdgeInsets.only(bottom: 8), child: Text('处理模式', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93)))),
                CupertinoSlidingSegmentedControl<int>(
                    groupValue: mode.value,
                    children: const {1: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('字幕', style: TextStyle(fontSize: 13))), 2: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('翻译', style: TextStyle(fontSize: 13))), 3: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('配音', style: TextStyle(fontSize: 13)))},
                    onValueChanged: (v) => mode.value = v!),
                const SizedBox(height: 20),
                const Padding(padding: EdgeInsets.only(bottom: 8), child: Text('目标语言', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93)))),
                MacosTextField(controller: langController, placeholder: 'zh (默认中文)', padding: const EdgeInsets.all(12), decoration: BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8), border: Border.all(color: const Color(0xFFE5E5EA)))),
              ],
            ),
          ),
        ],
      ),
    );
  }

  void _showInfo(BuildContext context, String msg) {
    showCupertinoDialog(
        context: context,
        builder: (_) => CupertinoAlertDialog(content: Text(msg), actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))]));
  }
}
