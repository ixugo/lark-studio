import 'dart:ui';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/task.dart';
import '../../../../providers.dart';
import '../../../core/glass_card.dart';
import '../../task/view_models/task_list_view_model.dart';
import '../../task/views/task_detail_page.dart';
import '../view_models/dashboard_view_model.dart';

class DashboardPage extends HookConsumerWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final backend = ref.watch(backendProvider);
    useListenable(backend);
    final ds = ref.watch(dashboardProvider);

    useEffect(() {
      Future.microtask(
          () => ref.read(dashboardProvider.notifier).load());
      return null;
    }, const []);

    return Column(
      children: [
        _buildToolbar(() => ref.read(dashboardProvider.notifier).load()),
        Expanded(
          child: ds.loading && !backend.online
              ? const Center(child: CupertinoActivityIndicator(radius: 14))
              : ListView(
                  padding: const EdgeInsets.fromLTRB(28, 20, 28, 28),
                  children: [
                    _buildWorkflowCards(context, ref),
                    const SizedBox(height: 24),
                    _buildRecentTasks(context, ds),
                    const SizedBox(height: 20),
                    _buildEnvReadiness(backend, ds),
                  ],
                ),
        ),
      ],
    );
  }

  Widget _buildWorkflowCards(BuildContext context, WidgetRef ref) {
    return Row(
      children: builtInWorkflows.map((wf) {
        final colors = wf.gradientColors
            .map((c) => Color(int.parse('FF$c', radix: 16)))
            .toList();
        return Expanded(
          child: Padding(
            padding: EdgeInsets.only(
                left: wf == builtInWorkflows.first ? 0 : 7,
                right: wf == builtInWorkflows.last ? 0 : 7),
            child: _WorkflowCard(
              title: wf.title,
              subtitle: wf.subtitle,
              steps: wf.steps,
              gradient: colors,
              onTap: wf.mode > 0
                  ? () => _startWorkflow(context, wf, ref)
                  : () => _showCustomWorkflow(context),
            ),
          ),
        );
      }).toList(),
    );
  }

  void _startWorkflow(
      BuildContext context, WorkflowTemplate wf, WidgetRef ref) {
    Navigator.of(context).push(CupertinoPageRoute(
      builder: (_) => _WorkflowDetailPage(workflow: wf),
    ));
  }

  void _showCustomWorkflow(BuildContext context) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('自定义流程'),
        content: const Text('自定义流程功能即将推出'),
        actions: [
          CupertinoDialogAction(
              child: const Text('确定'),
              onPressed: () => Navigator.pop(context))
        ],
      ),
    );
  }

  Widget _buildRecentTasks(BuildContext context, DashboardState ds) {
    return GlassCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 12),
            child: Row(
              children: [
                Container(
                  width: 4,
                  height: 16,
                  decoration: BoxDecoration(
                    color: const Color(0xFF007AFF),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                const SizedBox(width: 8),
                const Text('最近任务',
                    style: TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                        color: Color(0xFFE5E5EA),
                        letterSpacing: -0.3)),
              ],
            ),
          ),
          if (ds.recentTasks.isEmpty)
            const Padding(
              padding: EdgeInsets.all(32),
              child: Center(
                  child: Text('暂无任务',
                      style:
                          TextStyle(fontSize: 13, color: Color(0xFFC7C7CC)))),
            )
          else
            ...ds.recentTasks.map((t) => _RecentTaskRow(
                task: t,
                onTap: () {
                  Navigator.of(context).push(CupertinoPageRoute(
                    builder: (_) => TaskDetailPage(taskId: t.id),
                  ));
                })),
          const SizedBox(height: 10),
        ],
      ),
    );
  }

  Widget _buildEnvReadiness(dynamic backend, DashboardState ds) {
    final cfg = ds.config;
    final rows = <_EnvRow>[
      _EnvRow('引擎', backend.online,
          backend.online ? '端口 ${backend.port}' : '未启动'),
      _EnvRow('语音识别', cfg != null, cfg?.pipeline.whisperMode ?? '-'),
      _EnvRow('翻译服务', cfg?.llm.baseUrl.isNotEmpty == true,
          cfg?.llm.model ?? '未配置'),
      _EnvRow('配音服务', cfg != null,
          cfg?.tts.type == 'edge' ? 'Edge TTS' : 'OpenAI TTS'),
    ];
    final allReady = rows.every((r) => r.ready);

    return GlassCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 12),
            child: Row(
              children: [
                Container(
                  width: 4,
                  height: 16,
                  decoration: BoxDecoration(
                    color: allReady
                        ? const Color(0xFF34C759)
                        : const Color(0xFFFF9500),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                const SizedBox(width: 8),
                const Text('环境状态',
                    style: TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                        color: Color(0xFFE5E5EA),
                        letterSpacing: -0.3)),
                const Spacer(),
                if (allReady) _ReadyBadge(),
              ],
            ),
          ),
          ...rows.map((r) => _envRowWidget(r)),
          const SizedBox(height: 12),
        ],
      ),
    );
  }

  Widget _envRowWidget(_EnvRow row) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 7),
      child: Row(
        children: [
          SizedBox(
            width: 56,
            child: Text(row.label,
                style:
                    const TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
          ),
          Container(
            width: 7,
            height: 7,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: row.ready
                  ? const Color(0xFF34C759)
                  : const Color(0xFFC7C7CC),
              boxShadow: row.ready
                  ? [
                      BoxShadow(
                          color: const Color(0xFF34C759)
                              .withValues(alpha: 0.35),
                          blurRadius: 6)
                    ]
                  : null,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(row.value,
                style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: Color(0xFFE5E5EA)),
                overflow: TextOverflow.ellipsis),
          ),
        ],
      ),
    );
  }
}

// ---- 工作流卡片 ----

class _WorkflowCard extends HookWidget {
  final String title;
  final String subtitle;
  final List<WorkflowStep> steps;
  final List<Color> gradient;
  final VoidCallback onTap;

  const _WorkflowCard({
    required this.title,
    required this.subtitle,
    required this.steps,
    required this.gradient,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: ClipRRect(
          borderRadius: BorderRadius.circular(16),
          child: BackdropFilter(
            filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
            child: AnimatedContainer(
              duration: const Duration(milliseconds: 200),
              curve: Curves.easeOut,
              padding:
                  const EdgeInsets.symmetric(vertical: 20, horizontal: 18),
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [
                    gradient[0]
                        .withValues(alpha: hovering.value ? 0.20 : 0.12),
                    gradient.length > 1
                        ? gradient[1]
                            .withValues(alpha: hovering.value ? 0.10 : 0.05)
                        : gradient[0]
                            .withValues(alpha: hovering.value ? 0.10 : 0.05),
                  ],
                ),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(
                  color: gradient[0]
                      .withValues(alpha: hovering.value ? 0.25 : 0.12),
                  width: 0.5,
                ),
                boxShadow: hovering.value
                    ? [
                        BoxShadow(
                            color: gradient[0].withValues(alpha: 0.12),
                            blurRadius: 16,
                            offset: const Offset(0, 4))
                      ]
                    : null,
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title,
                      style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w700,
                          color: gradient[0],
                          letterSpacing: -0.3)),
                  const SizedBox(height: 4),
                  Text(subtitle,
                      style: TextStyle(
                          fontSize: 11,
                          color: gradient[0].withValues(alpha: 0.6))),
                  if (steps.isNotEmpty) ...[
                    const SizedBox(height: 14),
                    ...steps.asMap().entries.map((e) {
                      final idx = e.key;
                      final step = e.value;
                      return Padding(
                        padding: const EdgeInsets.symmetric(vertical: 3),
                        child: Row(
                          children: [
                            Container(
                              width: 18,
                              height: 18,
                              decoration: BoxDecoration(
                                shape: BoxShape.circle,
                                color:
                                    gradient[0].withValues(alpha: 0.15),
                              ),
                              child: Center(
                                child: Text('${idx + 1}',
                                    style: TextStyle(
                                        fontSize: 10,
                                        fontWeight: FontWeight.w600,
                                        color: gradient[0])),
                              ),
                            ),
                            const SizedBox(width: 8),
                            Text(step.label,
                                style: TextStyle(
                                    fontSize: 12,
                                    color: gradient[0]
                                        .withValues(alpha: 0.8))),
                          ],
                        ),
                      );
                    }),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

// ---- 工作流详情页 ----

class _WorkflowDetailPage extends HookConsumerWidget {
  final WorkflowTemplate workflow;
  const _WorkflowDetailPage({required this.workflow});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final creating = useState(false);
    final wf = workflow;
    final color =
        Color(int.parse('FF${wf.gradientColors[0]}', radix: 16));

    Future<void> pickAndCreate() async {
      final notifier = ref.read(taskListProvider.notifier);
      try {
        final result =
            await FilePicker.platform.pickFiles(type: FileType.video);
        if (result?.files.first.path == null) return;
        creating.value = true;
        await notifier.createTask(
            inputPath: result!.files.first.path!, mode: wf.mode);
        if (context.mounted) Navigator.pop(context);
      } catch (e) {
        if (context.mounted) {
          showCupertinoDialog(
              context: context,
              builder: (_) => CupertinoAlertDialog(
                    content: Text('创建失败: $e'),
                    actions: [
                      CupertinoDialogAction(
                          child: const Text('确定'),
                          onPressed: () => Navigator.pop(context))
                    ],
                  ));
        }
      }
      creating.value = false;
    }

    return CupertinoPageScaffold(
      backgroundColor: const Color(0xFF0A0A0A),
      navigationBar: CupertinoNavigationBar(
          middle: Text(wf.title), previousPageTitle: '主页'),
      child: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(28),
          children: [
            Text(wf.title,
                style: const TextStyle(
                    fontSize: 28,
                    fontWeight: FontWeight.w800,
                    color: Color(0xFFE5E5EA),
                    letterSpacing: -0.8)),
            const SizedBox(height: 6),
            Text(wf.subtitle,
                style: const TextStyle(
                    fontSize: 14, color: Color(0xFF8E8E93))),
            const SizedBox(height: 28),
            const Text('工作流步骤',
                style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFF8E8E93),
                    letterSpacing: 0.3)),
            const SizedBox(height: 12),
            ...wf.steps.asMap().entries.map((e) {
              final idx = e.key;
              final step = e.value;
              final isLast = idx == wf.steps.length - 1;
              return _StepRow(
                index: idx + 1,
                label: step.label,
                color: color,
                isLast: isLast,
              );
            }),
            const SizedBox(height: 32),
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.08),
                borderRadius: BorderRadius.circular(20),
              ),
              child: Icon(CupertinoIcons.cloud_upload,
                  size: 32, color: color.withValues(alpha: 0.7)),
            ),
            const SizedBox(height: 16),
            const Text('选择视频文件开始处理',
                style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                    color: Color(0xFFE5E5EA))),
            const SizedBox(height: 6),
            const Text('支持 MP4、MKV、AVI、MOV 等格式',
                style:
                    TextStyle(fontSize: 13, color: Color(0xFF8E8E93))),
            const SizedBox(height: 20),
            CupertinoButton.filled(
              borderRadius: BorderRadius.circular(12),
              onPressed: creating.value ? null : pickAndCreate,
              child: Text(
                  creating.value ? '创建中...' : '选择文件并开始',
                  style:
                      const TextStyle(fontWeight: FontWeight.w600)),
            ),
          ],
        ),
      ),
    );
  }
}

class _StepRow extends StatelessWidget {
  final int index;
  final String label;
  final Color color;
  final bool isLast;

  const _StepRow({
    required this.index,
    required this.label,
    required this.color,
    required this.isLast,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Column(
          children: [
            Container(
              width: 28,
              height: 28,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: color.withValues(alpha: 0.12),
                border: Border.all(
                    color: color.withValues(alpha: 0.3), width: 1.5),
              ),
              child: Center(
                child: Text('$index',
                    style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                        color: color)),
              ),
            ),
            if (!isLast)
              Container(
                width: 1.5,
                height: 24,
                color: color.withValues(alpha: 0.15),
              ),
          ],
        ),
        const SizedBox(width: 12),
        Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(label,
              style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w500,
                  color: Color(0xFFE5E5EA))),
        ),
      ],
    );
  }
}

// ---- 公用组件 ----

class _EnvRow {
  final String label;
  final bool ready;
  final String value;
  const _EnvRow(this.label, this.ready, this.value);
}

class _ReadyBadge extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: const Color(0xFF34C759).withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(8),
        border:
            Border.all(color: const Color(0xFF34C759).withValues(alpha: 0.2)),
      ),
      child: const Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(CupertinoIcons.checkmark_circle_fill,
              size: 11, color: Color(0xFF34C759)),
          SizedBox(width: 4),
          Text('全部就绪',
              style: TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w600,
                  color: Color(0xFF34C759))),
        ],
      ),
    );
  }
}

  Widget _buildToolbar(VoidCallback onRefresh) {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 28),
      decoration: const BoxDecoration(
        color: Color(0xFF0D0D0D),
        border: Border(bottom: BorderSide(color: Color(0xFF1F1F1F), width: 0.5)),
      ),
      child: Row(
        children: [
          const Text('启动台',
              style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: Color(0xFFE5E5EA),
                  letterSpacing: -0.5)),
          const Spacer(),
          CupertinoButton(
            padding: EdgeInsets.zero,
            onPressed: onRefresh,
            child: const Icon(CupertinoIcons.refresh,
                size: 16, color: Color(0xFF8E8E93)),
          ),
        ],
      ),
    );
  }

class _RecentTaskRow extends HookWidget {
  final Task task;
  final VoidCallback onTap;

  const _RecentTaskRow({required this.task, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 11),
          decoration: BoxDecoration(
            color: hovering.value
                ? const Color(0xFF007AFF).withValues(alpha: 0.04)
                : CupertinoColors.transparent,
            border: const Border(
                bottom: BorderSide(color: Color(0x0A000000), width: 0.5)),
          ),
          child: Row(
            children: [
              _statusDot(task.status),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(task.fileName,
                        style: const TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w500,
                            color: Color(0xFFE5E5EA)),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis),
                    const SizedBox(height: 1),
                    Text(task.modeName,
                        style: const TextStyle(
                            fontSize: 11, color: Color(0xFF8E8E93))),
                  ],
                ),
              ),
              Text(task.statusName,
                  style: TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w500,
                      color: _statusColor(task.status))),
              const SizedBox(width: 6),
              const Icon(CupertinoIcons.chevron_right,
                  size: 11, color: Color(0xFFC7C7CC)),
            ],
          ),
        ),
      ),
    );
  }

  Widget _statusDot(int status) {
    final color = _statusColor(status);
    return Container(
      width: 8,
      height: 8,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: color,
        boxShadow: [
          BoxShadow(color: color.withValues(alpha: 0.3), blurRadius: 4)
        ],
      ),
    );
  }

  Color _statusColor(int status) {
    switch (status) {
      case 1:
        return const Color(0xFFFF9500);
      case 2:
        return const Color(0xFFFFCC00);
      case 3:
        return const Color(0xFF34C759);
      case 4:
        return const Color(0xFFFF3B30);
      default:
        return const Color(0xFFC7C7CC);
    }
  }
}
