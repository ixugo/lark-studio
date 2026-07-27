import 'dart:ui';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../models/config.dart';
import '../models/task.dart';
import '../services/api_client.dart';
import '../services/backend_service.dart';
import 'task_detail_page.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key});

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  AppConfig? _config;
  List<Task> _recentTasks = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      final api = context.read<ApiClient>();
      final results = await Future.wait([
        api.getConfig(),
        api.listTasks(size: 5),
      ]);
      if (mounted) {
        setState(() {
          _config = results[0] as AppConfig;
          _recentTasks = results[1] as List<Task>;
          _loading = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final backend = context.watch<BackendService>();

    return Column(
      children: [
        _DashboardToolbar(onRefresh: _load),
        Expanded(
          child: _loading && !backend.online
              ? const Center(child: CupertinoActivityIndicator(radius: 14))
              : ListView(
                  padding: const EdgeInsets.fromLTRB(28, 20, 28, 28),
                  children: [
                    _buildQuickActions(),
                    const SizedBox(height: 24),
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(child: _buildRecentTasks()),
                        const SizedBox(width: 20),
                        SizedBox(width: 260, child: _buildEnvReadiness(backend)),
                      ],
                    ),
                  ],
                ),
        ),
      ],
    );
  }

  Widget _buildQuickActions() {
    return Row(
      children: [
        _ActionCard(
          icon: CupertinoIcons.film,
          label: '生成字幕',
          subtitle: 'Whisper 识别',
          gradient: const [Color(0xFF007AFF), Color(0xFF5AC8FA)],
          onTap: () => _quickCreate(1),
        ),
        const SizedBox(width: 14),
        _ActionCard(
          icon: CupertinoIcons.textformat,
          label: '翻译字幕',
          subtitle: 'LLM 翻译',
          gradient: const [Color(0xFFAF52DE), Color(0xFFDA8FFF)],
          onTap: () => _quickCreate(2),
        ),
        const SizedBox(width: 14),
        _ActionCard(
          icon: CupertinoIcons.mic_fill,
          label: '翻译配音',
          subtitle: 'TTS 合成',
          gradient: const [Color(0xFFFF9500), Color(0xFFFFCC00)],
          onTap: () => _quickCreate(3),
        ),
      ],
    );
  }

  void _quickCreate(int mode) {
    Navigator.of(context).push(CupertinoPageRoute(
      builder: (_) => _CreateTaskSheet(mode: mode),
    ));
  }

  Widget _buildRecentTasks() {
    return GlassCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 16, 18, 12),
            child: Row(
              children: [
                Container(
                  width: 4, height: 16,
                  decoration: BoxDecoration(
                    color: const Color(0xFF007AFF),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                const SizedBox(width: 8),
                const Text('最近任务', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F), letterSpacing: -0.3)),
              ],
            ),
          ),
          if (_recentTasks.isEmpty)
            const Padding(
              padding: EdgeInsets.all(32),
              child: Center(child: Text('暂无任务', style: TextStyle(fontSize: 13, color: Color(0xFFC7C7CC)))),
            )
          else
            ..._recentTasks.map((t) => _RecentTaskRow(task: t, onTap: () {
              Navigator.of(context).push(CupertinoPageRoute(
                builder: (_) => TaskDetailPage(taskId: t.id),
              ));
            })),
          const SizedBox(height: 10),
        ],
      ),
    );
  }

  Widget _buildEnvReadiness(BackendService backend) {
    final rows = <_EnvRow>[
      _EnvRow('引擎', backend.online, backend.online ? '端口 ${backend.port}' : '未启动'),
      _EnvRow('语音识别', _config != null, _config?.pipeline.whisperMode ?? '-'),
      _EnvRow('翻译服务', _config?.llm.baseUrl.isNotEmpty == true, _config?.llm.model ?? '未配置'),
      _EnvRow('配音服务', _config != null, _config?.tts.type == 'edge' ? 'Edge TTS' : 'OpenAI TTS'),
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
                  width: 4, height: 16,
                  decoration: BoxDecoration(
                    color: allReady ? const Color(0xFF34C759) : const Color(0xFFFF9500),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                const SizedBox(width: 8),
                const Text('环境状态', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F), letterSpacing: -0.3)),
                const Spacer(),
                if (allReady)
                  _ReadyBadge(),
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
            child: Text(row.label, style: const TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
          ),
          Container(
            width: 7, height: 7,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: row.ready ? const Color(0xFF34C759) : const Color(0xFFC7C7CC),
              boxShadow: row.ready
                  ? [BoxShadow(color: const Color(0xFF34C759).withValues(alpha: 0.35), blurRadius: 6)]
                  : null,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(row.value,
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF3A3A3C)),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}

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
        border: Border.all(color: const Color(0xFF34C759).withValues(alpha: 0.2)),
      ),
      child: const Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(CupertinoIcons.checkmark_circle_fill, size: 11, color: Color(0xFF34C759)),
          SizedBox(width: 4),
          Text('全部就绪', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: Color(0xFF34C759))),
        ],
      ),
    );
  }
}

class _DashboardToolbar extends StatelessWidget {
  final VoidCallback onRefresh;
  const _DashboardToolbar({required this.onRefresh});

  @override
  Widget build(BuildContext context) {
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          height: 52,
          padding: const EdgeInsets.symmetric(horizontal: 28),
          decoration: BoxDecoration(
            color: const Color(0xFFFFFFFF).withValues(alpha: 0.65),
            border: const Border(bottom: BorderSide(color: Color(0x1A000000), width: 0.5)),
          ),
          child: Row(
            children: [
              const Text('vdub', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800, color: Color(0xFF1D1D1F), letterSpacing: -0.8)),
              const SizedBox(width: 8),
              const Text('视频翻译配音', style: TextStyle(fontSize: 12, color: Color(0xFF8E8E93), letterSpacing: 0.1)),
              const Spacer(),
              CupertinoButton(
                padding: EdgeInsets.zero,
                onPressed: onRefresh,
                child: const Icon(CupertinoIcons.refresh, size: 16, color: Color(0xFF8E8E93)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 液态玻璃卡片
class GlassCard extends StatelessWidget {
  final Widget child;
  const GlassCard({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                const Color(0xFFFFFFFF).withValues(alpha: 0.82),
                const Color(0xFFF9F9FB).withValues(alpha: 0.72),
              ],
            ),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: const Color(0xFFFFFFFF).withValues(alpha: 0.5),
              width: 0.5,
            ),
            boxShadow: [
              BoxShadow(color: const Color(0xFF000000).withValues(alpha: 0.04), blurRadius: 16, offset: const Offset(0, 4)),
              BoxShadow(color: const Color(0xFF000000).withValues(alpha: 0.02), blurRadius: 4, offset: const Offset(0, 1)),
            ],
          ),
          child: child,
        ),
      ),
    );
  }
}

/// 快捷操作卡片——渐变玻璃风格
class _ActionCard extends StatefulWidget {
  final IconData icon;
  final String label;
  final String subtitle;
  final List<Color> gradient;
  final VoidCallback onTap;

  const _ActionCard({
    required this.icon,
    required this.label,
    required this.subtitle,
    required this.gradient,
    required this.onTap,
  });

  @override
  State<_ActionCard> createState() => _ActionCardState();
}

class _ActionCardState extends State<_ActionCard> {
  bool _hovering = false;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: MouseRegion(
        onEnter: (_) => setState(() => _hovering = true),
        onExit: (_) => setState(() => _hovering = false),
        child: GestureDetector(
          onTap: widget.onTap,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(16),
            child: BackdropFilter(
              filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 200),
                curve: Curves.easeOut,
                padding: const EdgeInsets.symmetric(vertical: 22, horizontal: 18),
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topLeft,
                    end: Alignment.bottomRight,
                    colors: [
                      widget.gradient[0].withValues(alpha: _hovering ? 0.20 : 0.12),
                      widget.gradient[1].withValues(alpha: _hovering ? 0.10 : 0.05),
                    ],
                  ),
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(
                    color: widget.gradient[0].withValues(alpha: _hovering ? 0.25 : 0.12),
                    width: 0.5,
                  ),
                  boxShadow: _hovering
                      ? [BoxShadow(color: widget.gradient[0].withValues(alpha: 0.12), blurRadius: 16, offset: const Offset(0, 4))]
                      : null,
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      width: 36, height: 36,
                      decoration: BoxDecoration(
                        color: widget.gradient[0].withValues(alpha: 0.12),
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Icon(widget.icon, size: 20, color: widget.gradient[0]),
                    ),
                    const SizedBox(height: 14),
                    Text(widget.label, style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: widget.gradient[0], letterSpacing: -0.2)),
                    const SizedBox(height: 2),
                    Text(widget.subtitle, style: TextStyle(fontSize: 11, color: widget.gradient[0].withValues(alpha: 0.55))),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// 最近任务行
class _RecentTaskRow extends StatefulWidget {
  final Task task;
  final VoidCallback onTap;

  const _RecentTaskRow({required this.task, required this.onTap});

  @override
  State<_RecentTaskRow> createState() => _RecentTaskRowState();
}

class _RecentTaskRowState extends State<_RecentTaskRow> {
  bool _hovering = false;

  @override
  Widget build(BuildContext context) {
    return MouseRegion(
      onEnter: (_) => setState(() => _hovering = true),
      onExit: (_) => setState(() => _hovering = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 11),
          decoration: BoxDecoration(
            color: _hovering ? const Color(0xFF007AFF).withValues(alpha: 0.04) : CupertinoColors.transparent,
            border: const Border(bottom: BorderSide(color: Color(0x0A000000), width: 0.5)),
          ),
          child: Row(
            children: [
              _statusDot(widget.task.status),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(widget.task.fileName,
                      style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF1D1D1F)),
                      maxLines: 1, overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 1),
                    Text(widget.task.modeName, style: const TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
                  ],
                ),
              ),
              Text(widget.task.statusName, style: TextStyle(fontSize: 11, fontWeight: FontWeight.w500, color: _statusColor(widget.task.status))),
              const SizedBox(width: 6),
              const Icon(CupertinoIcons.chevron_right, size: 11, color: Color(0xFFC7C7CC)),
            ],
          ),
        ),
      ),
    );
  }

  Widget _statusDot(int status) {
    final color = _statusColor(status);
    return Container(
      width: 8, height: 8,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: color,
        boxShadow: [BoxShadow(color: color.withValues(alpha: 0.3), blurRadius: 4)],
      ),
    );
  }

  Color _statusColor(int status) {
    switch (status) {
      case 1: return const Color(0xFFFF9500);
      case 2: return const Color(0xFFFFCC00);
      case 3: return const Color(0xFF34C759);
      case 4: return const Color(0xFFFF3B30);
      default: return const Color(0xFFC7C7CC);
    }
  }
}

/// 快捷创建任务页面
class _CreateTaskSheet extends StatefulWidget {
  final int mode;
  const _CreateTaskSheet({required this.mode});

  @override
  State<_CreateTaskSheet> createState() => _CreateTaskSheetState();
}

class _CreateTaskSheetState extends State<_CreateTaskSheet> {
  bool _creating = false;

  String get _modeName {
    switch (widget.mode) {
      case 1: return '生成字幕';
      case 2: return '翻译字幕';
      case 3: return '翻译配音';
      default: return '处理';
    }
  }

  @override
  Widget build(BuildContext context) {
    return CupertinoPageScaffold(
      backgroundColor: const Color(0xFFF5F5F7),
      navigationBar: CupertinoNavigationBar(middle: Text(_modeName), previousPageTitle: '首页'),
      child: SafeArea(
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 72, height: 72,
                decoration: BoxDecoration(
                  color: const Color(0xFF007AFF).withValues(alpha: 0.08),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Icon(CupertinoIcons.cloud_upload, size: 32, color: const Color(0xFF007AFF).withValues(alpha: 0.7)),
              ),
              const SizedBox(height: 20),
              const Text('选择视频文件', style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F), letterSpacing: -0.3)),
              const SizedBox(height: 6),
              const Text('支持 MP4、MKV、AVI、MOV 等格式', style: TextStyle(fontSize: 13, color: Color(0xFF8E8E93))),
              const SizedBox(height: 28),
              CupertinoButton.filled(
                borderRadius: BorderRadius.circular(12),
                onPressed: _creating ? null : _pickAndCreate,
                child: Text(_creating ? '创建中...' : '选择文件并开始', style: const TextStyle(fontWeight: FontWeight.w600)),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _pickAndCreate() async {
    final notifier = context.read<TaskListNotifier>();
    final result = await _pickFile();
    if (result == null) return;
    setState(() => _creating = true);
    try {
      await notifier.createTask(inputPath: result, mode: widget.mode);
      if (mounted) Navigator.pop(context);
    } catch (e) {
      if (mounted) {
        showCupertinoDialog(context: context, builder: (_) => CupertinoAlertDialog(
          content: Text('创建失败: $e'),
          actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))],
        ));
      }
    }
    if (mounted) setState(() => _creating = false);
  }

  Future<String?> _pickFile() async {
    try {
      final result = await FilePicker.platform.pickFiles(type: FileType.video);
      return result?.files.first.path;
    } catch (_) {
      return null;
    }
  }
}
