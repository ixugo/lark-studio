import 'dart:async';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../models/task.dart';
import '../services/api_client.dart';
import '../services/websocket_service.dart';
import 'create_task_page.dart';
import 'task_detail_page.dart';

class TaskListPage extends StatefulWidget {
  const TaskListPage({super.key});

  @override
  State<TaskListPage> createState() => _TaskListPageState();
}

class _TaskListPageState extends State<TaskListPage> {
  StreamSubscription? _wsSub;

  @override
  void initState() {
    super.initState();
    final notifier = context.read<TaskListNotifier>();
    final ws = context.read<WebSocketService>();
    Future.microtask(() {
      notifier.refresh();
      _wsSub = ws.events.listen((event) {
        if (event.type.startsWith('task_') && mounted) {
          notifier.refresh();
        }
      });
    });
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final notifier = context.watch<TaskListNotifier>();
    return Column(
      children: [
        _buildToolbar(context),
        Expanded(child: _buildBody(context, notifier)),
      ],
    );
  }

  Widget _buildToolbar(BuildContext context) {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 20),
      decoration: const BoxDecoration(
        color: Color(0xFFFAFAFA),
        border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5)),
      ),
      child: Row(
        children: [
          const Text('任务列表', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
          const Spacer(),
          CupertinoButton(
            padding: EdgeInsets.zero,
            onPressed: () => context.read<TaskListNotifier>().refresh(),
            child: const Icon(CupertinoIcons.arrow_clockwise, size: 18),
          ),
          const SizedBox(width: 8),
          CupertinoButton(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            color: const Color(0xFFE5E5EA),
            borderRadius: BorderRadius.circular(8),
            onPressed: () => _showBatchDialog(context),
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(CupertinoIcons.folder_badge_plus, size: 16, color: Color(0xFF3A3A3C)),
                SizedBox(width: 4),
                Text('批量导入', style: TextStyle(fontSize: 13, color: Color(0xFF3A3A3C))),
              ],
            ),
          ),
          const SizedBox(width: 8),
          CupertinoButton.filled(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
            borderRadius: BorderRadius.circular(8),
            onPressed: () => _showCreateDialog(context),
            child: const Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(CupertinoIcons.add, size: 16, color: CupertinoColors.white),
                SizedBox(width: 4),
                Text('新建任务', style: TextStyle(fontSize: 13, color: CupertinoColors.white)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context, TaskListNotifier notifier) {
    if (notifier.loading && notifier.tasks.isEmpty) {
      return const Center(child: CupertinoActivityIndicator(radius: 14));
    }
    if (notifier.error != null && notifier.tasks.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(CupertinoIcons.exclamationmark_circle, size: 48, color: Color(0xFFC7C7CC)),
            const SizedBox(height: 12),
            Text(notifier.error!, style: const TextStyle(color: Color(0xFF8E8E93), fontSize: 14)),
            const SizedBox(height: 16),
            CupertinoButton(onPressed: () => notifier.refresh(), child: const Text('重试')),
          ],
        ),
      );
    }
    if (notifier.tasks.isEmpty) {
      return const Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(CupertinoIcons.film, size: 56, color: Color(0xFFC7C7CC)),
            SizedBox(height: 12),
            Text('暂无任务', style: TextStyle(color: Color(0xFF8E8E93), fontSize: 15)),
            SizedBox(height: 4),
            Text('点击右上角新建任务开始使用', style: TextStyle(color: Color(0xFFC7C7CC), fontSize: 13)),
          ],
        ),
      );
    }

    return ListView.separated(
      padding: const EdgeInsets.all(20),
      itemCount: notifier.tasks.length,
      separatorBuilder: (_, _) => const SizedBox(height: 8),
      itemBuilder: (context, index) {
        final t = notifier.tasks[index];
        return _TaskCard(
          task: t,
          onDelete: () => _confirmDelete(context, t),
          onPause: t.canPause ? () => notifier.pauseTask(t.id) : null,
          onResume: t.canResume ? () => notifier.resumeTask(t.id) : null,
        );
      },
    );
  }

  void _showCreateDialog(BuildContext context) {
    showCupertinoModalPopup(context: context, builder: (_) => const CreateTaskPage());
  }

  void _showBatchDialog(BuildContext context) {
    showCupertinoModalPopup(context: context, builder: (_) => const _BatchImportSheet());
  }

  void _confirmDelete(BuildContext context, Task task) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('删除任务'),
        content: Text('确定删除 "${task.fileName}" 吗？'),
        actions: [
          CupertinoDialogAction(isDefaultAction: true, child: const Text('取消'), onPressed: () => Navigator.pop(context)),
          CupertinoDialogAction(
            isDestructiveAction: true,
            child: const Text('删除'),
            onPressed: () {
              Navigator.pop(context);
              context.read<TaskListNotifier>().deleteTask(task.id);
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
  const _TaskCard({required this.task, required this.onDelete, this.onPause, this.onResume});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () {
        Navigator.of(context).push(CupertinoPageRoute(builder: (_) => TaskDetailPage(taskId: task.id)));
      },
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: CupertinoColors.white,
          borderRadius: BorderRadius.circular(10),
          boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 2))],
        ),
        child: Row(
          children: [
            _statusIcon(task.status),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(task.fileName, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500, color: Color(0xFF1D1D1F)),
                    maxLines: 1, overflow: TextOverflow.ellipsis),
                  const SizedBox(height: 4),
                  Row(
                    children: [
                      _badge(task.modeName, CupertinoColors.systemBlue),
                      const SizedBox(width: 8),
                      _badge(task.statusName, _statusColor(task.status)),
                      if (task.currentStep.isNotEmpty) ...[
                        const SizedBox(width: 8),
                        Text(task.currentStep, style: const TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
                      ],
                    ],
                  ),
                ],
              ),
            ),
            if (onPause != null)
              CupertinoButton(
                padding: EdgeInsets.zero,
                minimumSize: const Size(28, 28),
                onPressed: onPause,
                child: const Icon(CupertinoIcons.pause_circle, size: 20, color: CupertinoColors.systemOrange),
              ),
            if (onResume != null)
              CupertinoButton(
                padding: EdgeInsets.zero,
                minimumSize: const Size(28, 28),
                onPressed: onResume,
                child: const Icon(CupertinoIcons.play_circle, size: 20, color: CupertinoColors.systemGreen),
              ),
            CupertinoButton(
              padding: EdgeInsets.zero,
              minimumSize: const Size(28, 28),
              onPressed: onDelete,
              child: const Icon(CupertinoIcons.trash, size: 16, color: Color(0xFFC7C7CC)),
            ),
            const SizedBox(width: 4),
            const Icon(CupertinoIcons.chevron_right, size: 14, color: Color(0xFFC7C7CC)),
          ],
        ),
      ),
    );
  }

  // 0=待处理, 1=进行中, 2=已暂停, 3=已完成, 4=失败
  Widget _statusIcon(int status) {
    switch (status) {
      case 1:
        return const CupertinoActivityIndicator(radius: 10);
      case 2:
        return const Icon(CupertinoIcons.pause_circle_fill, color: CupertinoColors.systemYellow, size: 22);
      case 3:
        return const Icon(CupertinoIcons.checkmark_circle_fill, color: CupertinoColors.systemGreen, size: 22);
      case 4:
        return const Icon(CupertinoIcons.xmark_circle_fill, color: CupertinoColors.systemRed, size: 22);
      default:
        return const Icon(CupertinoIcons.clock, color: Color(0xFFC7C7CC), size: 22);
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
      decoration: BoxDecoration(color: color.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(4)),
      child: Text(text, style: TextStyle(fontSize: 11, color: color, fontWeight: FontWeight.w500)),
    );
  }
}

class _BatchImportSheet extends StatefulWidget {
  const _BatchImportSheet();

  @override
  State<_BatchImportSheet> createState() => _BatchImportSheetState();
}

class _BatchImportSheetState extends State<_BatchImportSheet> {
  final _langController = TextEditingController(text: 'zh');
  final List<String> _selectedFiles = [];
  int _mode = 3;
  bool _submitting = false;

  @override
  void dispose() {
    _langController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
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
            decoration: const BoxDecoration(
              border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5)),
            ),
            child: Row(
              children: [
                CupertinoButton(padding: EdgeInsets.zero, onPressed: () => Navigator.pop(context),
                  child: const Text('取消', style: TextStyle(fontSize: 15))),
                const Expanded(child: Text('批量导入', textAlign: TextAlign.center,
                  style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F)))),
                CupertinoButton(padding: EdgeInsets.zero,
                  onPressed: (_submitting || _selectedFiles.isEmpty) ? null : _submit,
                  child: Text('导入 (${_selectedFiles.length})', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600,
                    color: (_submitting || _selectedFiles.isEmpty) ? const Color(0xFFC7C7CC) : CupertinoColors.systemBlue))),
              ],
            ),
          ),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.all(20),
              children: [
                Row(
                  children: [
                    const Text('已选视频', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93))),
                    const Spacer(),
                    CupertinoButton(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                      color: CupertinoColors.systemBlue, borderRadius: BorderRadius.circular(8),
                      onPressed: _pickFiles,
                      child: const Text('选择视频文件', style: TextStyle(color: CupertinoColors.white, fontSize: 13))),
                  ],
                ),
                const SizedBox(height: 8),
                if (_selectedFiles.isEmpty)
                  Container(
                    padding: const EdgeInsets.all(24),
                    decoration: BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0xFFE5E5EA))),
                    child: const Center(child: Text('点击右上角选择视频文件', style: TextStyle(fontSize: 13, color: Color(0xFFC7C7CC)))),
                  )
                else
                  Container(
                    constraints: const BoxConstraints(maxHeight: 150),
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0xFFE5E5EA))),
                    child: ListView.builder(
                      shrinkWrap: true,
                      itemCount: _selectedFiles.length,
                      itemBuilder: (_, i) {
                        final name = _selectedFiles[i].split('/').last;
                        return Padding(padding: const EdgeInsets.symmetric(vertical: 3),
                          child: Row(children: [
                            const Icon(CupertinoIcons.film, size: 14, color: Color(0xFF8E8E93)),
                            const SizedBox(width: 6),
                            Expanded(child: Text(name, style: const TextStyle(fontSize: 12, color: Color(0xFF3A3A3C)),
                              maxLines: 1, overflow: TextOverflow.ellipsis)),
                            GestureDetector(
                              onTap: () => setState(() => _selectedFiles.removeAt(i)),
                              child: const Icon(CupertinoIcons.xmark_circle_fill, size: 16, color: Color(0xFFC7C7CC)),
                            ),
                          ]));
                      },
                    ),
                  ),
                const SizedBox(height: 20),
                const Padding(padding: EdgeInsets.only(bottom: 8),
                  child: Text('处理模式', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93)))),
                CupertinoSlidingSegmentedControl<int>(groupValue: _mode, children: const {
                  1: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('字幕', style: TextStyle(fontSize: 13))),
                  2: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('翻译', style: TextStyle(fontSize: 13))),
                  3: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('配音', style: TextStyle(fontSize: 13))),
                }, onValueChanged: (v) => setState(() => _mode = v!)),
                const SizedBox(height: 20),
                const Padding(padding: EdgeInsets.only(bottom: 8),
                  child: Text('目标语言', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93)))),
                CupertinoTextField(controller: _langController, placeholder: 'zh (默认中文)',
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: const Color(0xFFE5E5EA)))),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _pickFiles() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.video, allowMultiple: true, dialogTitle: '选择视频文件');
    if (result != null) {
      setState(() {
        for (final f in result.files) {
          if (f.path != null && !_selectedFiles.contains(f.path)) {
            _selectedFiles.add(f.path!);
          }
        }
      });
    }
  }

  Future<void> _submit() async {
    setState(() => _submitting = true);
    try {
      final count = await context.read<TaskListNotifier>().batchCreateTasks(
        videos: List.from(_selectedFiles), mode: _mode, targetLang: _langController.text.trim());
      if (mounted) {
        Navigator.pop(context);
        _showInfo('已创建 $count 个任务');
      }
    } on ApiException catch (e) {
      if (mounted) _showInfo(e.message);
    } catch (e) {
      if (mounted) _showInfo('批量创建失败: $e');
    }
    if (mounted) setState(() => _submitting = false);
  }

  void _showInfo(String msg) {
    showCupertinoDialog(context: context, builder: (_) => CupertinoAlertDialog(
      content: Text(msg),
      actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))],
    ));
  }
}
