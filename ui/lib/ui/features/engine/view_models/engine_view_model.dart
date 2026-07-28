import 'dart:async';
import 'dart:ui' show Color;
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart' show IconData;
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../data/repositories/config_repository.dart';
import '../../../../data/services/websocket_service.dart';
import '../../../../providers.dart';

/// 引擎定义
class EngineItem {
  final String id;
  final String name;
  final String category;
  final IconData icon;
  final Color color;
  final String description;
  final bool available;
  final List<String> tags;

  const EngineItem({
    required this.id,
    required this.name,
    required this.category,
    this.icon = CupertinoIcons.bolt,
    this.color = const Color(0xFF8E8E93),
    this.description = '',
    this.available = false,
    this.tags = const [],
  });

  static const empty = EngineItem(id: '', name: '', category: '');
}

/// 引擎页状态
class EngineState {
  final List<EngineItem> engines;
  final String? selectedEngineId;
  final String whisperMode;
  final String whisperModel;
  final Map<String, dynamic> whisperRuntime;
  final bool installing;
  final String installMessage;

  const EngineState({
    this.engines = const [],
    this.selectedEngineId,
    this.whisperMode = '',
    this.whisperModel = '',
    this.whisperRuntime = const {},
    this.installing = false,
    this.installMessage = '',
  });

  EngineState copyWith({
    List<EngineItem>? engines,
    String? selectedEngineId,
    bool clearSelection = false,
    String? whisperMode,
    String? whisperModel,
    Map<String, dynamic>? whisperRuntime,
    bool? installing,
    String? installMessage,
  }) {
    return EngineState(
      engines: engines ?? this.engines,
      selectedEngineId: clearSelection
          ? null
          : (selectedEngineId ?? this.selectedEngineId),
      whisperMode: whisperMode ?? this.whisperMode,
      whisperModel: whisperModel ?? this.whisperModel,
      whisperRuntime: whisperRuntime ?? this.whisperRuntime,
      installing: installing ?? this.installing,
      installMessage: installMessage ?? this.installMessage,
    );
  }
}

/// 引擎状态管理
class EngineNotifier extends Notifier<EngineState> {
  late final ConfigRepository _configRepo;

  @override
  EngineState build() {
    _configRepo = ref.watch(configRepoProvider);
    final subscription = ref
        .watch(wsServiceProvider)
        .events
        .listen(_handleRuntimeEvent);
    ref.onDispose(subscription.cancel);
    return const EngineState();
  }

  /// 处理运行时安装输出，并在安装结束后重取真实状态。
  void _handleRuntimeEvent(WsEvent event) {
    switch (event.type) {
      case 'whisper_runtime_log':
        state = state.copyWith(
          installing: true,
          installMessage: event.data['message'] as String? ?? '',
        );
      case 'whisper_runtime_done':
        state = state.copyWith(
          whisperRuntime: event.data,
          installing: false,
          installMessage: '安装完成',
        );
        unawaited(load());
      case 'whisper_runtime_failed':
        state = state.copyWith(
          installing: false,
          installMessage: event.data['error'] as String? ?? '安装失败',
        );
    }
  }

  /// 加载引擎配置
  Future<void> load() async {
    try {
      final config = await _configRepo.getConfig();
      final runtime = await _configRepo.getWhisperRuntime();
      final pipeline = config.pipeline;
      final runtimeAvailable = runtime['installed'] as bool? ?? false;

      final engines = <EngineItem>[
        EngineItem(
          id: 'whisper_cpp',
          name: 'whisper.cpp（内置）',
          category: 'local',
          icon: CupertinoIcons.waveform,
          color: const Color(0xFF34C759),
          description: '高性能本地语音识别引擎，支持 Apple Metal 加速',
          available: runtimeAvailable,
          tags: const ['Apple 芯片', 'Metal', '本地'],
        ),
        const EngineItem(
          id: 'faster_whisper',
          name: 'faster-whisper',
          category: 'local',
          icon: CupertinoIcons.bolt,
          color: Color(0xFFFF9500),
          description: 'NVIDIA GPU 优化的 Whisper 推理引擎',
          tags: ['NVIDIA', '高速', '高精度'],
        ),
        const EngineItem(
          id: 'funasr',
          name: 'FunASR',
          category: 'local_multi',
          icon: CupertinoIcons.waveform_path,
          color: Color(0xFF5AC8FA),
          description: '阿里达摩院语音识别模型',
        ),
        const EngineItem(
          id: 'qwen3_asr',
          name: 'Qwen3-ASR',
          category: 'local_multi',
          icon: CupertinoIcons.waveform_path,
          color: Color(0xFF007AFF),
          description: '通义千问语音识别模型',
        ),
        const EngineItem(
          id: 'custom_cmd',
          name: '本地命令行',
          category: 'local_cmd',
          icon: CupertinoIcons.command,
          color: Color(0xFF8E8E93),
          description: '自定义命令行引擎调用',
          tags: ['自定义命令', '高级'],
        ),
        const EngineItem(
          id: 'openai_asr',
          name: 'OpenAI',
          category: 'cloud',
          icon: CupertinoIcons.cloud,
          color: Color(0xFF10A37F),
          description: 'OpenAI Whisper API 云端服务',
        ),
        const EngineItem(
          id: 'groq',
          name: 'Groq',
          category: 'cloud',
          icon: CupertinoIcons.cloud,
          color: Color(0xFFFF6B6B),
          description: 'Groq 超高速推理服务',
        ),
        const EngineItem(
          id: 'deepgram',
          name: 'Deepgram',
          category: 'cloud',
          icon: CupertinoIcons.cloud,
          color: Color(0xFF5856D6),
          description: '实时语音转写云服务',
        ),
      ];

      state = state.copyWith(
        engines: engines,
        whisperMode: pipeline.whisperMode,
        whisperModel: pipeline.whisperModel,
        whisperRuntime: runtime,
        installing: runtime['installing'] as bool? ?? false,
      );
    } catch (_) {}
  }

  /// 安装本地 Whisper 运行时，实际进度由 WebSocket 事件更新。
  Future<void> installWhisper() async {
    if (state.installing) return;
    state = state.copyWith(installing: true, installMessage: '准备安装…');
    try {
      await _configRepo.installWhisperRuntime();
    } catch (error) {
      state = state.copyWith(installing: false, installMessage: '安装失败：$error');
    }
  }

  /// 选中引擎
  void selectEngine(String? id) {
    state = state.copyWith(selectedEngineId: id, clearSelection: id == null);
  }
}

final engineProvider = NotifierProvider<EngineNotifier, EngineState>(
  EngineNotifier.new,
);
