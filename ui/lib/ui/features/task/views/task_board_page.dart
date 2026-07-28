import 'dart:async';
import 'dart:io';

import 'package:flutter/cupertino.dart' show CupertinoColors, CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../../data/models/task.dart';
import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../../core/glass_card.dart';
import '../view_models/task_list_view_model.dart';

/// 任务主从视图：左侧任务列表，右侧选中任务的步骤链与实时日志。
/// 数据层复用 taskListProvider / wsService / apiClient，与任务列表页同源。
class TaskBoardPage extends HookConsumerWidget {
  const TaskBoardPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final c = AppColors.of(context);
    final ts = ref.watch(taskListProvider);
    final notifier = ref.read(taskListProvider.notifier);
    final ws = ref.read(wsServiceProvider);
    final api = ref.read(apiClientProvider);

    final selectedId = useState<String?>(null);
    final detail = useState<Task?>(null);
    final logs = useState(<TaskLog>[]);
    final steps = useState(<TaskStep>[]);
    final requestGeneration = useRef(0);

    /// loadDetail 并行读取任务、步骤和日志，并丢弃已切换任务的旧响应。
    Future<void> loadDetail(String id) async {
      final generation = ++requestGeneration.value;
      try {
        final result = await Future.wait([
          api.getTask(id),
          api.getTaskLogs(id),
          api.listTaskSteps(id),
        ]);
        if (generation != requestGeneration.value || selectedId.value != id) {
          return;
        }
        detail.value = result[0] as Task;
        logs.value = result[1] as List<TaskLog>;
        steps.value = result[2] as List<TaskStep>;
      } catch (e) {
        if (generation != requestGeneration.value || selectedId.value != id) {
          return;
        }
        logs.value = [...logs.value, TaskLog.local(id, '加载任务详情失败：$e')];
      }
    }

    /// refreshDetail 只刷新高频变化的任务与步骤，避免反复读取历史日志。
    Future<void> refreshDetail(String id) async {
      final generation = ++requestGeneration.value;
      try {
        final result = await Future.wait([
          api.getTask(id),
          api.listTaskSteps(id),
        ]);
        if (generation != requestGeneration.value || selectedId.value != id) {
          return;
        }
        detail.value = result[0] as Task;
        steps.value = result[1] as List<TaskStep>;
      } catch (error) {
        debugPrint('刷新任务详情失败：$error');
      }
    }

    void select(String id) {
      selectedId.value = id;
      detail.value = null;
      logs.value = [];
      steps.value = [];
      loadDetail(id);
    }

    // WebSocket 日志即时追加，进度事件合并刷新，避免高频请求阻塞界面。
    useEffect(() {
      Future.microtask(() => notifier.refresh());
      Timer? refreshTimer;
      final sub = ws.events.listen((event) {
        if (!event.type.startsWith('task_')) return;
        final sid = selectedId.value;
        final eventTaskId = event.data['task_id'] as String? ?? '';
        if (event.type == 'task_log' && sid == eventTaskId) {
          final log = TaskLog.fromJson(event.data);
          if (log.message.isEmpty ||
              logs.value.any((item) => item.id == log.id)) {
            return;
          }
          final next = [...logs.value, log];
          logs.value = next.length > 1000
              ? next.sublist(next.length - 1000)
              : next;
          return;
        }
        refreshTimer?.cancel();
        refreshTimer = Timer(const Duration(milliseconds: 220), () {
          notifier.refresh();
          if (sid != null && sid == eventTaskId) {
            refreshDetail(sid);
          }
        });
      });
      return () {
        refreshTimer?.cancel();
        unawaited(sub.cancel());
      };
    }, const []);

    // 列表就绪后默认选中首个任务
    useEffect(() {
      if (selectedId.value == null && ts.tasks.isNotEmpty) {
        select(ts.tasks.first.id);
      }
      return null;
    }, [ts.tasks]);

    /// deleteTask 先清理详情状态，再删除任务，避免已销毁页面保留遮罩。
    void deleteTask(Task t) {
      selectedId.value = null;
      detail.value = null;
      logs.value = [];
      steps.value = [];
      notifier.deleteTask(t.id);
    }

    // 全面屏主从布局：左侧任务列表通高，右侧详情无卡片边距
    return Row(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SizedBox(
          width: 300,
          child: GlassBar(
            padding: const EdgeInsets.all(8),
            border: Border(right: BorderSide(color: c.glassStroke, width: 0.5)),
            child: _buildList(c, ts, selectedId.value, select),
          ),
        ),
        Expanded(
          child: _buildDetail(
            c,
            detail.value,
            steps.value,
            logs.value,
            notifier,
            deleteTask,
          ),
        ),
      ],
    );
  }

  Widget _buildList(
    AppColors c,
    TaskListState ts,
    String? selectedId,
    ValueChanged<String> onSelect,
  ) {
    if (ts.loading && ts.tasks.isEmpty) {
      return const Center(child: ProgressCircle(radius: 14));
    }
    if (ts.tasks.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(CupertinoIcons.tray, size: 40, color: c.textQuaternary),
            const SizedBox(height: 10),
            Text('暂无任务', style: TextStyle(fontSize: 14, color: c.textTertiary)),
          ],
        ),
      );
    }
    return ListView.builder(
      itemCount: ts.tasks.length,
      itemBuilder: (_, i) {
        final t = ts.tasks[i];
        return _TaskCell(
          task: t,
          selected: t.id == selectedId,
          onTap: () => onSelect(t.id),
        );
      },
    );
  }

  Widget _buildDetail(
    AppColors c,
    Task? t,
    List<TaskStep> steps,
    List<TaskLog> logs,
    TaskListNotifier notifier,
    void Function(Task) onDelete,
  ) {
    if (t == null) {
      return Center(
        child: Text(
          '选择左侧任务查看详情',
          style: TextStyle(fontSize: 13, color: c.textTertiary),
        ),
      );
    }
    // 固定布局：头部与步骤链定高，日志盒占满余量，避免嵌套滚动视图
    return Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _DetailHeader(task: t, notifier: notifier, onDelete: onDelete),
          const SizedBox(height: 16),
          _SectionLabel('处理进度', c),
          _StepChain(task: t, history: steps),
          const SizedBox(height: 16),
          _SectionLabel('实时日志', c),
          Expanded(
            child: _LogBox(logs: logs, task: t, c: c),
          ),
        ],
      ),
    );
  }
}

/// 列表单元：状态点 + 文件名 + 模式/状态 + 运行中进度条
class _TaskCell extends HookWidget {
  final Task task;
  final bool selected;
  final VoidCallback onTap;
  const _TaskCell({
    required this.task,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final hovering = useState(false);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          margin: const EdgeInsets.only(bottom: 4),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: selected
                ? c.accent.withValues(alpha: 0.16)
                : hovering.value
                ? c.glassCardHover
                : const Color(0x00000000),
            borderRadius: BorderRadius.circular(10),
            border: selected
                ? Border.all(color: c.accent.withValues(alpha: 0.4), width: 0.5)
                : null,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  _StatusDot(status: task.status),
                  const SizedBox(width: 9),
                  Expanded(
                    child: Text(
                      task.fileName,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w500,
                        color: c.textPrimary,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
              Padding(
                padding: const EdgeInsets.only(left: 17, top: 4),
                child: Text(
                  task.status == 1
                      ? '${task.modeName} · ${task.relativeCreatedAt()}'
                      : '${task.modeName} · ${task.statusName} · ${task.relativeCreatedAt()}',
                  style: TextStyle(fontSize: 11.5, color: c.textTertiary),
                ),
              ),
              if (task.status == 1)
                Padding(
                  padding: const EdgeInsets.only(left: 17, top: 8),
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(2),
                    child: SizedBox(
                      height: 4,
                      child: Stack(
                        children: [
                          Container(color: c.inputBg),
                          FractionallySizedBox(
                            widthFactor: (task.progress.clamp(0, 100)) / 100,
                            child: Container(color: c.accent),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 状态点以呼吸光晕强调仍在运行的任务，静态状态保持克制。
class _StatusDot extends HookWidget {
  final int status;
  const _StatusDot({required this.status});

  @override
  Widget build(BuildContext context) {
    final pulse = useAnimationController(
      duration: const Duration(milliseconds: 900),
    );
    useEffect(() {
      if (status == 1) {
        pulse.repeat(reverse: true);
      } else {
        pulse.stop();
        pulse.value = 0;
      }
      return null;
    }, [status]);
    final color = switch (status) {
      1 => AppColors.of(context).accent,
      2 => AppColors.yellow,
      3 => AppColors.green,
      4 => AppColors.red,
      _ => AppColors.of(context).textQuaternary,
    };
    return AnimatedBuilder(
      animation: pulse,
      builder: (_, _) => Container(
        width: 8,
        height: 8,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: color,
          boxShadow: status == 1
              ? [
                  BoxShadow(
                    color: color.withValues(alpha: 0.28 + pulse.value * 0.44),
                    blurRadius: 4 + pulse.value * 7,
                    spreadRadius: pulse.value,
                  ),
                ]
              : null,
        ),
      ),
    );
  }
}

/// 详情头部：文件名 + 模式/状态徽标 + 暂停/恢复/删除
class _DetailHeader extends HookWidget {
  final Task task;
  final TaskListNotifier notifier;
  final void Function(Task) onDelete;
  const _DetailHeader({
    required this.task,
    required this.notifier,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final confirmingDelete = useState(false);
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                task.fileName,
                style: TextStyle(
                  fontSize: 17,
                  fontWeight: FontWeight.w700,
                  color: c.textPrimary,
                  letterSpacing: -0.2,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                '${task.modeName} · 创建于 ${task.relativeCreatedAt()}',
                style: TextStyle(fontSize: 12, color: c.textTertiary),
              ),
            ],
          ),
        ),
        if (task.canPause) ...[
          AppButton.secondary(
            color: AppColors.orange,
            onPressed: () => notifier.pauseTask(task.id),
            child: const Text('暂停'),
          ),
          const SizedBox(width: 8),
        ],
        if (task.canResume) ...[
          AppButton(
            color: AppColors.green,
            onPressed: () => notifier.resumeTask(task.id),
            child: Text(task.status == 4 ? '重试' : '恢复'),
          ),
          const SizedBox(width: 8),
        ],
        AppButton.secondary(
          onPressed: () => _openFolder(task.outputDir),
          child: Icon(CupertinoIcons.folder, size: 14, color: c.textSecondary),
        ),
        const SizedBox(width: 8),
        if (confirmingDelete.value) ...[
          AppButton.secondary(
            onPressed: () => confirmingDelete.value = false,
            child: const Text('取消'),
          ),
          const SizedBox(width: 6),
          AppButton(
            color: AppColors.red,
            onPressed: () => onDelete(task),
            child: const Text('确认删除'),
          ),
        ] else
          AppButton.secondary(
            onPressed: () => confirmingDelete.value = true,
            child: Icon(CupertinoIcons.trash, size: 14, color: AppColors.red),
          ),
      ],
    );
  }

  /// _openFolder 调用 macOS Finder 打开任务输出目录。
  Future<void> _openFolder(String path) async {
    if (path.isEmpty) return;
    try {
      await Process.start('open', [path]);
    } catch (error) {
      debugPrint('打开输出目录失败：$error');
    }
  }
}

/// 步骤链以连线表达处理顺序，并突出当前正在执行的步骤。
class _StepChain extends StatelessWidget {
  final Task task;
  final List<TaskStep> history;
  const _StepChain({required this.task, required this.history});

  static const _labels = {
    'whisper': '听写',
    'split': '语义分句',
    'translate': '翻译',
    'tts': '配音',
    'merge': '音频合成',
    'lipsync': '对口型',
    'burn': '视频烧录',
  };

  @override
  Widget build(BuildContext context) {
    final records = {for (final item in history) item.name: item};
    final hasLipSync =
        records.containsKey('lipsync') || task.currentStep == 'lipsync';
    final stepNames = task.mode == 3
        ? [
            'whisper',
            'split',
            'translate',
            'tts',
            'merge',
            if (hasLipSync) 'lipsync',
            'burn',
          ]
        : task.mode == 2
        ? ['whisper', 'split', 'translate', 'burn']
        : ['whisper', 'burn'];
    final currentIndex = stepNames.indexOf(task.currentStep);

    return Column(
      children: stepNames.asMap().entries.map((entry) {
        final name = entry.value;
        final record = records[name];
        final active = task.status == 1 && name == task.currentStep;
        final failed =
            record?.status == 3 ||
            (task.status == 4 && name == task.currentStep);
        final done =
            task.status == 3 ||
            record?.status == 2 ||
            record?.status == 4 ||
            (currentIndex >= 0 && entry.key < currentIndex);
        final progress = active
            ? task.stepProgress
            : record?.progress ?? (done ? 100 : 0);
        final detail = active ? task.currentDetail : record?.detail ?? '';
        return _PipelineStep(
          label: _labels[name] ?? taskStepName(name),
          meta: _stepMeta(record, detail, progress, active, failed),
          progress: progress,
          done: done,
          active: active,
          failed: failed,
          hasNext: entry.key < stepNames.length - 1,
        );
      }).toList(),
    );
  }

  /// _stepMeta 组合步骤右侧的模型、进度、耗时与失败信息。
  String _stepMeta(
    TaskStep? record,
    String detail,
    int progress,
    bool active,
    bool failed,
  ) {
    if (failed) {
      final reason = record?.error ?? '';
      return reason.isEmpty ? '失败' : reason;
    }
    if (active) {
      return detail.isEmpty ? '$progress%' : '$detail · $progress%';
    }
    if (record?.status == 4) {
      return detail.isEmpty ? '已跳过' : '$detail · 已跳过';
    }
    final elapsed = record?.elapsedText ?? '';
    if (detail.isEmpty) return elapsed;
    return elapsed.isEmpty ? detail : '$detail · $elapsed';
  }
}

/// 单个步骤在运行时播放呼吸动画，使进度在静态列表中保持可见。
class _PipelineStep extends HookWidget {
  final String label;
  final String meta;
  final int progress;
  final bool done;
  final bool active;
  final bool failed;
  final bool hasNext;

  const _PipelineStep({
    required this.label,
    required this.meta,
    required this.progress,
    required this.done,
    required this.active,
    required this.failed,
    required this.hasNext,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final pulse = useAnimationController(
      duration: const Duration(milliseconds: 1100),
    );
    useEffect(() {
      if (active) {
        pulse.repeat(reverse: true);
      } else {
        pulse.stop();
        pulse.value = 0;
      }
      return null;
    }, [active]);

    final color = failed
        ? AppColors.red
        : active
        ? c.accent
        : done
        ? AppColors.green
        : c.textQuaternary;
    final progressFactor = progress.clamp(0, 100) / 100;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 36,
            child: Column(
              children: [
                AnimatedBuilder(
                  animation: pulse,
                  builder: (_, _) => Container(
                    width: 28,
                    height: 28,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: done || failed ? color : c.bg,
                      border: Border.all(color: color, width: active ? 2.5 : 2),
                      boxShadow: active
                          ? [
                              BoxShadow(
                                color: color.withValues(
                                  alpha: 0.16 + pulse.value * 0.24,
                                ),
                                blurRadius: 7 + pulse.value * 8,
                                spreadRadius: pulse.value * 2,
                              ),
                            ]
                          : null,
                    ),
                    child: done
                        ? Icon(
                            CupertinoIcons.checkmark,
                            size: 16,
                            color: CupertinoColors.white,
                          )
                        : failed
                        ? Icon(
                            CupertinoIcons.xmark,
                            size: 15,
                            color: CupertinoColors.white,
                          )
                        : active
                        ? Center(
                            child: Container(
                              width: 10,
                              height: 10,
                              decoration: BoxDecoration(
                                color: color,
                                shape: BoxShape.circle,
                              ),
                            ),
                          )
                        : null,
                  ),
                ),
                if (hasNext)
                  Container(
                    width: 1.5,
                    height: active ? 29 : 20,
                    color: active
                        ? c.accent.withValues(alpha: 0.5)
                        : c.glassStroke,
                  ),
              ],
            ),
          ),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.only(top: 4, right: 4),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text(
                        label,
                        style: TextStyle(
                          fontSize: 14,
                          color: active || failed ? color : c.textPrimary,
                          fontWeight: active || failed
                              ? FontWeight.w600
                              : FontWeight.w500,
                        ),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Text(
                          meta,
                          textAlign: TextAlign.right,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 12,
                            color: active ? c.accent : c.textTertiary,
                            fontWeight: active
                                ? FontWeight.w500
                                : FontWeight.w400,
                          ),
                        ),
                      ),
                    ],
                  ),
                  if (active) ...[
                    const SizedBox(height: 8),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(2),
                      child: Container(
                        height: 3,
                        color: c.inputBg,
                        child: TweenAnimationBuilder<double>(
                          tween: Tween(end: progressFactor),
                          duration: const Duration(milliseconds: 260),
                          curve: Curves.easeOutCubic,
                          builder: (_, value, _) => FractionallySizedBox(
                            alignment: Alignment.centerLeft,
                            widthFactor: value,
                            child: Container(color: c.accent),
                          ),
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  final String text;
  final AppColors c;
  const _SectionLabel(this.text, this.c);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 13,
          fontWeight: FontWeight.w600,
          color: c.textSecondary,
        ),
      ),
    );
  }
}

/// 日志盒：等宽字体、半透明黑底、自动停底
class _LogBox extends StatelessWidget {
  final List<TaskLog> logs;
  final Task task;
  final AppColors c;
  const _LogBox({required this.logs, required this.task, required this.c});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: c.isDark ? const Color(0x66000000) : const Color(0x0A000000),
        borderRadius: BorderRadius.circular(9),
        border: Border.all(color: c.glassStroke, width: 0.5),
      ),
      child: logs.isEmpty
          ? Center(
              child: Text(
                task.status == 1 ? '正在等待实时输出…' : '该任务尚无实时日志',
                style: TextStyle(fontSize: 12, color: c.textTertiary),
              ),
            )
          : ListView.builder(
              reverse: true,
              itemCount: logs.length,
              itemBuilder: (_, i) {
                final idx = logs.length - 1 - i;
                return Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Text(
                    '[${logs[idx].timeText}] ${logs[idx].message}',
                    style: TextStyle(
                      fontSize: 12,
                      fontFamily: 'monospace',
                      height: 1.35,
                      color: _logColor(logs[idx]),
                    ),
                  ),
                );
              },
            ),
    );
  }

  /// _logColor 用颜色区分成功、警告、错误与进度日志。
  Color _logColor(TaskLog log) {
    if (log.level == 'error') return AppColors.red;
    if (log.level == 'warn') return AppColors.orange;
    if (log.level == 'success') return AppColors.green;
    if (log.message.contains('进度')) return c.accent;
    return c.textSecondary;
  }
}
