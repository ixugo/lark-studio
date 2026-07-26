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
        _toolbar(),
        Expanded(
          child: _loading && !backend.online
              ? const Center(child: CupertinoActivityIndicator(radius: 14))
              : ListView(
                  padding: const EdgeInsets.all(24),
                  children: [
                    _buildQuickActions(),
                    const SizedBox(height: 20),
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Expanded(child: _buildRecentTasks()),
                        const SizedBox(width: 16),
                        SizedBox(width: 280, child: _buildEnvReadiness(backend)),
                      ],
                    ),
                  ],
                ),
        ),
      ],
    );
  }

  Widget _toolbar() {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 24),
      decoration: const BoxDecoration(
        color: Color(0xFFFAFAFA),
        border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5)),
      ),
      child: const Row(
        children: [
          Text('vdub', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: Color(0xFF1D1D1F), letterSpacing: -0.5)),
          SizedBox(width: 8),
          Text('视频翻译配音', style: TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
        ],
      ),
    );
  }

  Widget _buildQuickActions() {
    return Row(
      children: [
        _ActionCard(
          icon: CupertinoIcons.film,
          label: '生成字幕',
          color: CupertinoColors.systemBlue,
          onTap: () => _quickCreate(1),
        ),
        const SizedBox(width: 12),
        _ActionCard(
          icon: CupertinoIcons.textformat,
          label: '翻译字幕',
          color: CupertinoColors.systemPurple,
          onTap: () => _quickCreate(2),
        ),
        const SizedBox(width: 12),
        _ActionCard(
          icon: CupertinoIcons.mic_fill,
          label: '翻译配音',
          color: CupertinoColors.systemOrange,
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
    return _GlassCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 14, 16, 10),
            child: Text('最近任务', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
          ),
          if (_recentTasks.isEmpty)
            const Padding(
              padding: EdgeInsets.all(24),
              child: Center(child: Text('暂无任务', style: TextStyle(fontSize: 13, color: Color(0xFFC7C7CC)))),
            )
          else
            ..._recentTasks.map((t) => _RecentTaskRow(task: t, onTap: () {
              Navigator.of(context).push(CupertinoPageRoute(
                builder: (_) => TaskDetailPage(taskId: t.id),
              ));
            })),
          const SizedBox(height: 8),
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

    return _GlassCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 10),
            child: Row(
              children: [
                const Text('环境就绪', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
                const Spacer(),
                if (allReady)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: CupertinoColors.systemGreen.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: const Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(CupertinoIcons.checkmark_circle_fill, size: 12, color: CupertinoColors.systemGreen),
                        SizedBox(width: 4),
                        Text('全部就绪', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w500, color: CupertinoColors.systemGreen)),
                      ],
                    ),
                  ),
              ],
            ),
          ),
          ...rows.map((r) => _envRowWidget(r)),
          const SizedBox(height: 8),
        ],
      ),
    );
  }

  Widget _envRowWidget(_EnvRow row) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      child: Row(
        children: [
          SizedBox(
            width: 64,
            child: Text(row.label, style: const TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
          ),
          Container(
            width: 7, height: 7,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: row.ready ? CupertinoColors.systemGreen : const Color(0xFFC7C7CC),
              boxShadow: row.ready
                  ? [BoxShadow(color: CupertinoColors.systemGreen.withValues(alpha: 0.3), blurRadius: 4)]
                  : null,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(row.value, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF3A3A3C)),
              overflow: TextOverflow.ellipsis),
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

/// 毛玻璃卡片
class _GlassCard extends StatelessWidget {
  final Widget child;
  const _GlassCard({required this.child});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(14),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
        child: Container(
          decoration: BoxDecoration(
            color: CupertinoColors.white.withValues(alpha: 0.78),
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: const Color(0xFFE5E5EA).withValues(alpha: 0.6), width: 0.5),
            boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 12, offset: Offset(0, 4))],
          ),
          child: child,
        ),
      ),
    );
  }
}

/// 快捷操作卡片
class _ActionCard extends StatefulWidget {
  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;

  const _ActionCard({required this.icon, required this.label, required this.color, required this.onTap});

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
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            padding: const EdgeInsets.symmetric(vertical: 20, horizontal: 16),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft, end: Alignment.bottomRight,
                colors: [
                  widget.color.withValues(alpha: _hovering ? 0.18 : 0.10),
                  widget.color.withValues(alpha: _hovering ? 0.08 : 0.04),
                ],
              ),
              borderRadius: BorderRadius.circular(14),
              border: Border.all(color: widget.color.withValues(alpha: 0.15)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(widget.icon, size: 28, color: widget.color),
                const SizedBox(height: 10),
                Text(widget.label, style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: widget.color)),
                const SizedBox(height: 2),
                Text('选择视频开始', style: TextStyle(fontSize: 11, color: widget.color.withValues(alpha: 0.6))),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// 最近任务行
class _RecentTaskRow extends StatelessWidget {
  final Task task;
  final VoidCallback onTap;

  const _RecentTaskRow({required this.task, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        decoration: const BoxDecoration(
          border: Border(bottom: BorderSide(color: Color(0xFFF2F2F7), width: 0.5)),
        ),
        child: Row(
          children: [
            _statusDot(task.status),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(task.fileName, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF1D1D1F)),
                    maxLines: 1, overflow: TextOverflow.ellipsis),
                  Text(task.modeName, style: const TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
                ],
              ),
            ),
            Text(task.statusName, style: TextStyle(fontSize: 11, fontWeight: FontWeight.w500, color: _statusColor(task.status))),
            const SizedBox(width: 4),
            const Icon(CupertinoIcons.chevron_right, size: 12, color: Color(0xFFC7C7CC)),
          ],
        ),
      ),
    );
  }

  Widget _statusDot(int status) {
    return Container(
      width: 8, height: 8,
      decoration: BoxDecoration(shape: BoxShape.circle, color: _statusColor(status)),
    );
  }

  Color _statusColor(int status) {
    switch (status) {
      case 1: return CupertinoColors.systemOrange;
      case 2: return CupertinoColors.systemYellow;
      case 3: return CupertinoColors.systemGreen;
      case 4: return CupertinoColors.systemRed;
      default: return const Color(0xFFC7C7CC);
    }
  }
}

/// 快捷创建任务的简易页面
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
              Icon(CupertinoIcons.cloud_upload, size: 48, color: CupertinoColors.systemBlue.withValues(alpha: 0.6)),
              const SizedBox(height: 16),
              const Text('选择视频文件', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
              const SizedBox(height: 8),
              const Text('支持 MP4、MKV、AVI、MOV 等格式', style: TextStyle(fontSize: 13, color: Color(0xFF8E8E93))),
              const SizedBox(height: 24),
              CupertinoButton.filled(
                onPressed: _creating ? null : _pickAndCreate,
                child: Text(_creating ? '创建中...' : '选择文件并开始'),
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
