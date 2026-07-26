import 'dart:async';

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
    Future.microtask(() {
      context.read<TaskListNotifier>().refresh();
      _wsSub = context.read<WebSocketService>().events.listen((event) {
        if (event.type.startsWith('task_')) {
          context.read<TaskListNotifier>().refresh();
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
      separatorBuilder: (_, __) => const SizedBox(height: 8),
      itemBuilder: (context, index) => _TaskCard(
        task: notifier.tasks[index],
        onDelete: () => _confirmDelete(context, notifier.tasks[index]),
      ),
    );
  }

  void _showCreateDialog(BuildContext context) {
    showCupertinoModalPopup(context: context, builder: (_) => const CreateTaskPage());
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
  const _TaskCard({required this.task, required this.onDelete});

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
            CupertinoButton(
              padding: EdgeInsets.zero,
              minSize: 28,
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

  Widget _statusIcon(int status) {
    switch (status) {
      case 1:
        return const CupertinoActivityIndicator(radius: 10);
      case 2:
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
