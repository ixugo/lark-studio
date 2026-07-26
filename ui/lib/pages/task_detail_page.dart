import 'dart:async';

import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../models/task.dart';
import '../services/api_client.dart';
import '../services/websocket_service.dart';

class TaskDetailPage extends StatefulWidget {
  final String taskId;
  const TaskDetailPage({super.key, required this.taskId});

  @override
  State<TaskDetailPage> createState() => _TaskDetailPageState();
}

class _TaskDetailPageState extends State<TaskDetailPage> {
  Task? _task;
  String? _error;
  final List<String> _logs = [];
  StreamSubscription? _wsSub;

  @override
  void initState() {
    super.initState();
    _loadTask();
    _wsSub = context.read<WebSocketService>().events.listen((event) {
      final eventTaskId = event.data['task_id'] as String?;
      if (eventTaskId != widget.taskId) return;

      switch (event.type) {
        case 'task_log':
          final msg = event.data['message'] as String? ?? '';
          if (msg.isNotEmpty) setState(() => _logs.add(msg));
          break;
        case 'task_progress':
        case 'task_step_done':
        case 'task_done':
        case 'task_failed':
        case 'task_paused':
          _loadTask();
          break;
      }
    });
  }

  @override
  void dispose() {
    _wsSub?.cancel();
    super.dispose();
  }

  Future<void> _loadTask() async {
    try {
      final t = await context.read<ApiClient>().getTask(widget.taskId);
      if (mounted) setState(() { _task = t; _error = null; });
    } catch (e) {
      if (mounted) setState(() => _error = '$e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return CupertinoPageScaffold(
      backgroundColor: CupertinoColors.systemGroupedBackground,
      navigationBar: CupertinoNavigationBar(
        middle: Text(_task?.fileName ?? '任务详情'),
        previousPageTitle: '任务',
      ),
      child: SafeArea(child: _buildContent()),
    );
  }

  Widget _buildContent() {
    if (_task == null && _error == null) {
      return const Center(child: CupertinoActivityIndicator(radius: 14));
    }
    if (_task == null) {
      return Center(child: Text(_error ?? '加载失败', style: const TextStyle(color: Color(0xFF8E8E93))));
    }

    final t = _task!;
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        _infoCard([
          _row('文件', t.fileName),
          _row('模式', t.modeName),
          _row('状态', t.statusName),
          if (t.currentStep.isNotEmpty) _row('当前步骤', t.currentStep),
          _row('目标语言', t.targetLang),
          if (t.error.isNotEmpty) _row('错误', t.error),
        ]),
        if (t.canPause || t.canResume) ...[
          const SizedBox(height: 12),
          _buildActions(t),
        ],
        const SizedBox(height: 16),
        _infoCard([
          _row('输入路径', t.inputPath),
          _row('输出目录', t.outputDir),
          _row('创建时间', _formatTime(t.createdAt)),
          _row('更新时间', _formatTime(t.updatedAt)),
        ]),
        const SizedBox(height: 16),
        _buildProgress(t),
        if (_logs.isNotEmpty) ...[
          const SizedBox(height: 16),
          _buildLogs(),
        ],
      ],
    );
  }

  Widget _infoCard(List<Widget> children) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: CupertinoColors.white,
        borderRadius: BorderRadius.circular(10),
        boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 2))],
      ),
      child: Column(children: children),
    );
  }

  Widget _row(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(width: 80, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF8E8E93)))),
          Expanded(child: Text(value, style: const TextStyle(fontSize: 13, color: Color(0xFF1D1D1F)))),
        ],
      ),
    );
  }

  Widget _buildActions(Task t) {
    return Row(
      children: [
        if (t.canPause)
          Expanded(
            child: CupertinoButton(
              color: CupertinoColors.systemOrange,
              padding: const EdgeInsets.symmetric(vertical: 10),
              borderRadius: BorderRadius.circular(10),
              onPressed: _pausing ? null : _doPause,
              child: const Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(CupertinoIcons.pause_circle, size: 18, color: CupertinoColors.white),
                  SizedBox(width: 6),
                  Text('暂停', style: TextStyle(fontSize: 14, color: CupertinoColors.white)),
                ],
              ),
            ),
          ),
        if (t.canResume)
          Expanded(
            child: CupertinoButton(
              color: CupertinoColors.systemGreen,
              padding: const EdgeInsets.symmetric(vertical: 10),
              borderRadius: BorderRadius.circular(10),
              onPressed: _resuming ? null : _doResume,
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  const Icon(CupertinoIcons.play_circle, size: 18, color: CupertinoColors.white),
                  const SizedBox(width: 6),
                  Text(t.status == 4 ? '重试' : '恢复', style: const TextStyle(fontSize: 14, color: CupertinoColors.white)),
                ],
              ),
            ),
          ),
      ],
    );
  }

  bool _pausing = false;
  bool _resuming = false;

  Future<void> _doPause() async {
    setState(() => _pausing = true);
    try {
      await context.read<ApiClient>().pauseTask(widget.taskId);
      await _loadTask();
    } catch (e) {
      if (mounted) setState(() => _logs.add('暂停失败: $e'));
    }
    if (mounted) setState(() => _pausing = false);
  }

  Future<void> _doResume() async {
    setState(() => _resuming = true);
    try {
      await context.read<ApiClient>().resumeTask(widget.taskId);
      await _loadTask();
    } catch (e) {
      if (mounted) setState(() => _logs.add('恢复失败: $e'));
    }
    if (mounted) setState(() => _resuming = false);
  }

  Widget _buildProgress(Task t) {
    final steps = t.mode == 3
        ? ['whisper', 'split', 'translate', 'tts', 'merge', 'burn']
        : t.mode == 2
            ? ['whisper', 'split', 'translate', 'burn']
            : ['whisper', 'burn'];
    final currentIdx = steps.indexOf(t.currentStep);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: CupertinoColors.white,
        borderRadius: BorderRadius.circular(10),
        boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 2))],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('处理进度', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
          const SizedBox(height: 12),
          ...steps.asMap().entries.map((e) {
            final idx = e.key;
            final name = e.value;
            final isDone = (t.status == 3) || idx < currentIdx;
            final isActive = t.status == 1 && idx == currentIdx;
            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(
                children: [
                  if (isDone)
                    const Icon(CupertinoIcons.checkmark_circle_fill, size: 18, color: CupertinoColors.systemGreen)
                  else if (isActive)
                    const CupertinoActivityIndicator(radius: 8)
                  else
                    const Icon(CupertinoIcons.circle, size: 18, color: Color(0xFFD1D1D6)),
                  const SizedBox(width: 10),
                  Text(
                    _stepLabel(name),
                    style: TextStyle(
                      fontSize: 13,
                      color: isActive ? CupertinoColors.systemBlue : const Color(0xFF3A3A3C),
                      fontWeight: isActive ? FontWeight.w600 : FontWeight.w400,
                    ),
                  ),
                ],
              ),
            );
          }),
        ],
      ),
    );
  }

  Widget _buildLogs() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: CupertinoColors.white,
        borderRadius: BorderRadius.circular(10),
        boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 2))],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('实时日志', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
          const SizedBox(height: 8),
          Container(
            constraints: const BoxConstraints(maxHeight: 200),
            child: ListView.builder(
              reverse: true,
              shrinkWrap: true,
              itemCount: _logs.length,
              itemBuilder: (_, i) {
                final idx = _logs.length - 1 - i;
                return Padding(
                  padding: const EdgeInsets.symmetric(vertical: 2),
                  child: Text(_logs[idx], style: const TextStyle(fontSize: 12, fontFamily: 'monospace', color: Color(0xFF6E6E73))),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  String _stepLabel(String step) {
    const labels = {
      'whisper': '语音识别 (Whisper)',
      'split': '句级分段 (Split)',
      'translate': '翻译 (Translate)',
      'tts': '语音合成 (TTS)',
      'merge': '音频合并 (Merge)',
      'burn': '字幕烧录 (Burn)',
    };
    return labels[step] ?? step;
  }

  String _formatTime(DateTime dt) {
    return '${dt.year}-${_pad(dt.month)}-${_pad(dt.day)} ${_pad(dt.hour)}:${_pad(dt.minute)}:${_pad(dt.second)}';
  }

  String _pad(int n) => n.toString().padLeft(2, '0');
}
