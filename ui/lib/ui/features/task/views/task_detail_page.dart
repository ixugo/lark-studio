import 'dart:async';

import 'package:flutter/cupertino.dart' show CupertinoColors, CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/task.dart';
import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../../core/glass_card.dart';

class TaskDetailPage extends HookConsumerWidget {
  final String taskId;
  const TaskDetailPage({super.key, required this.taskId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final c = AppColors.of(context);
    final api = ref.read(apiClientProvider);
    final ws = ref.read(wsServiceProvider);
    final task = useState<Task?>(null);
    final error = useState<String?>(null);
    final logs = useState(<String>[]);
    final pausing = useState(false);
    final resuming = useState(false);

    Future<void> loadTask() async {
      try {
        final t = await api.getTask(taskId);
        task.value = t;
        error.value = null;
      } catch (e) {
        error.value = '$e';
      }
    }

    useEffect(() {
      loadTask();
      final sub = ws.events.listen((event) {
        final eventTaskId = event.data['task_id'] as String?;
        if (eventTaskId != taskId) return;
        switch (event.type) {
          case 'task_log':
            final msg = event.data['message'] as String? ?? '';
            if (msg.isNotEmpty) logs.value = [...logs.value, msg];
            break;
          case 'task_progress':
          case 'task_step_done':
          case 'task_done':
          case 'task_failed':
          case 'task_paused':
            loadTask();
            break;
        }
      });
      return sub.cancel;
    }, [taskId]);

    Future<void> doPause() async {
      pausing.value = true;
      try {
        await api.pauseTask(taskId);
        await loadTask();
      } catch (e) {
        logs.value = [...logs.value, '暂停失败: $e'];
      }
      pausing.value = false;
    }

    Future<void> doResume() async {
      resuming.value = true;
      try {
        await api.resumeTask(taskId);
        await loadTask();
      } catch (e) {
        logs.value = [...logs.value, '恢复失败: $e'];
      }
      resuming.value = false;
    }

    final t = task.value;
    return Container(
      color: c.contentBg,
      child: Column(
        children: [
          _DetailNavBar(title: t?.fileName ?? '任务详情', c: c),
          Expanded(
            child: _buildContent(context, c, t, error.value, logs.value,
                pausing.value, resuming.value, doPause, doResume),
          ),
        ],
      ),
    );
  }

  Widget _buildContent(
      BuildContext context,
      AppColors c,
      Task? t,
      String? err,
      List<String> logs,
      bool pausing,
      bool resuming,
      Future<void> Function() doPause,
      Future<void> Function() doResume) {
    if (t == null && err == null) {
      return const Center(child: ProgressCircle(radius: 14));
    }
    if (t == null) {
      return Center(
          child: Text(err ?? '加载失败',
              style: TextStyle(color: c.textTertiary)));
    }

    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        _infoCard(c, [
          _row(c, '文件', t.fileName),
          _row(c, '模式', t.modeName),
          _row(c, '状态', t.statusName),
          if (t.currentStep.isNotEmpty) _row(c, '当前步骤', t.currentStep),
          _row(c, '目标语言', t.targetLang),
          if (t.error.isNotEmpty) _row(c, '错误', t.error),
        ]),
        if (t.canPause || t.canResume) ...[
          const SizedBox(height: 12),
          _buildActions(t, pausing, resuming, doPause, doResume),
        ],
        const SizedBox(height: 16),
        _infoCard(c, [
          _row(c, '输入路径', t.inputPath),
          _row(c, '输出目录', t.outputDir),
          _row(c, '创建时间', _formatTime(t.createdAt)),
          _row(c, '更新时间', _formatTime(t.updatedAt)),
        ]),
        const SizedBox(height: 16),
        _buildProgress(c, t),
        if (logs.isNotEmpty) ...[
          const SizedBox(height: 16),
          _buildLogs(c, logs),
        ],
      ],
    );
  }

  Widget _infoCard(AppColors c, List<Widget> children) {
    return GlassCard(
      padding: const EdgeInsets.all(16),
      child: Column(children: children),
    );
  }

  Widget _row(AppColors c, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
        SizedBox(
            width: 80,
            child: Text(label,
                style: TextStyle(fontSize: 13, color: c.textTertiary))),
        Expanded(
            child: Text(value,
                style: TextStyle(fontSize: 13, color: c.textPrimary))),
      ]),
    );
  }

  Widget _buildActions(Task t, bool pausing, bool resuming,
      Future<void> Function() doPause, Future<void> Function() doResume) {
    return Row(children: [
      if (t.canPause)
        Expanded(
            child: AppButton(
                color: AppColors.orange,
                onPressed: pausing ? null : doPause,
                child: const Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(CupertinoIcons.pause_circle,
                          size: 18, color: Color(0xFFFFFFFF)),
                      SizedBox(width: 6),
                      Text('暂停', style: TextStyle(fontSize: 14)),
                    ]))),
      if (t.canResume)
        Expanded(
            child: AppButton(
                color: AppColors.green,
                onPressed: resuming ? null : doResume,
                child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      const Icon(CupertinoIcons.play_circle,
                          size: 18, color: Color(0xFFFFFFFF)),
                      const SizedBox(width: 6),
                      Text(t.status == 4 ? '重试' : '恢复',
                          style: const TextStyle(fontSize: 14)),
                    ]))),
    ]);
  }

  Widget _buildProgress(AppColors c, Task t) {
    final steps = t.mode == 3
        ? ['whisper', 'split', 'translate', 'tts', 'merge', 'burn']
        : t.mode == 2
            ? ['whisper', 'split', 'translate', 'burn']
            : ['whisper', 'burn'];
    final currentIdx = steps.indexOf(t.currentStep);

    return GlassCard(
      padding: const EdgeInsets.all(16),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('处理进度',
            style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: c.textPrimary)),
        const SizedBox(height: 12),
        ...steps.asMap().entries.map((e) {
          final idx = e.key;
          final name = e.value;
          final isDone = (t.status == 3) || idx < currentIdx;
          final isActive = t.status == 1 && idx == currentIdx;
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Row(children: [
              if (isDone)
                const Icon(CupertinoIcons.checkmark_circle_fill,
                    size: 18, color: CupertinoColors.systemGreen)
              else if (isActive)
                const ProgressCircle(radius: 8)
              else
                Icon(CupertinoIcons.circle,
                    size: 18, color: c.textQuaternary),
              const SizedBox(width: 10),
              Text(_stepLabel(name),
                  style: TextStyle(
                      fontSize: 13,
                      color: isActive ? c.accent : c.textPrimary,
                      fontWeight:
                          isActive ? FontWeight.w600 : FontWeight.w400)),
            ]),
          );
        }),
      ]),
    );
  }

  Widget _buildLogs(AppColors c, List<String> logs) {
    return GlassCard(
      padding: const EdgeInsets.all(16),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('实时日志',
            style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: c.textPrimary)),
        const SizedBox(height: 8),
        Container(
          height: 200,
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: c.isDark
                ? const Color(0x47FFFFFF)
                : const Color(0x0A000000),
            borderRadius: BorderRadius.circular(8),
          ),
          child: ListView.builder(
            reverse: true,
            itemCount: logs.length,
            itemBuilder: (_, i) {
              final idx = logs.length - 1 - i;
              return Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Text(logs[idx],
                    style: TextStyle(
                        fontSize: 12,
                        fontFamily: 'monospace',
                        color: c.textSecondary)),
              );
            },
          ),
        ),
      ]),
    );
  }

  String _stepLabel(String step) {
    const labels = {'whisper': '语音识别 (Whisper)', 'split': '句级分段 (Split)', 'translate': '翻译 (Translate)', 'tts': '语音合成 (TTS)', 'merge': '音频合并 (Merge)', 'burn': '字幕烧录 (Burn)'};
    return labels[step] ?? step;
  }

  String _formatTime(DateTime dt) => '${dt.year}-${_pad(dt.month)}-${_pad(dt.day)} ${_pad(dt.hour)}:${_pad(dt.minute)}:${_pad(dt.second)}';
  String _pad(int n) => n.toString().padLeft(2, '0');
}

/// 详情页顶部导航：返回箭头 + 标题，替代 CupertinoNavigationBar 以贴合全局玻璃风格
class _DetailNavBar extends HookWidget {
  final String title;
  final AppColors c;
  const _DetailNavBar({required this.title, required this.c});

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    return GlassBar(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      border: Border(bottom: BorderSide(color: c.glassStroke, width: 0.5)),
      child: Row(children: [
        MouseRegion(
          cursor: SystemMouseCursors.click,
          onEnter: (_) => hovering.value = true,
          onExit: (_) => hovering.value = false,
          child: GestureDetector(
            onTap: () => Navigator.of(context).pop(),
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 120),
              padding: const EdgeInsets.all(6),
              decoration: BoxDecoration(
                color: hovering.value
                    ? c.glassCardBg
                    : const Color(0x00000000),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(mainAxisSize: MainAxisSize.min, children: [
                Icon(CupertinoIcons.chevron_left,
                    size: 16, color: c.textSecondary),
                const SizedBox(width: 2),
                Text('任务',
                    style: TextStyle(fontSize: 13, color: c.textSecondary)),
              ]),
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(title,
              style: TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                  color: c.textPrimary),
              maxLines: 1,
              overflow: TextOverflow.ellipsis),
        ),
      ]),
    );
  }
}
