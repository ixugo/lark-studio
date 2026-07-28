import 'dart:ui' show Color;
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart' show IconData;
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/repositories/config_repository.dart';
import '../../../../data/models/config.dart';
import '../../../../providers.dart';

/// TTS 服务定义
class TTSService {
  final String id;
  final String name;
  final String category;
  final IconData icon;
  final Color color;
  final String description;
  final bool available;
  final String badge;

  const TTSService({
    required this.id,
    required this.name,
    required this.category,
    this.icon = CupertinoIcons.waveform,
    this.color = const Color(0xFF8E8E93),
    this.description = '',
    this.available = false,
    this.badge = '',
  });

  static const empty = TTSService(id: '', name: '', category: '');
}

/// TTS 页状态
class TTSState {
  final List<TTSService> services;
  final String? selectedServiceId;
  final String currentType;
  final String currentVoice;
  final TTSConfig config;
  final bool saving;

  const TTSState({
    this.services = const [],
    this.selectedServiceId,
    this.currentType = '',
    this.currentVoice = '',
    this.config = const TTSConfig(),
    this.saving = false,
  });

  TTSState copyWith({
    List<TTSService>? services,
    String? selectedServiceId,
    bool clearSelection = false,
    String? currentType,
    String? currentVoice,
    TTSConfig? config,
    bool? saving,
  }) {
    return TTSState(
      services: services ?? this.services,
      selectedServiceId: clearSelection
          ? null
          : (selectedServiceId ?? this.selectedServiceId),
      currentType: currentType ?? this.currentType,
      currentVoice: currentVoice ?? this.currentVoice,
      config: config ?? this.config,
      saving: saving ?? this.saving,
    );
  }
}

/// TTS 状态管理
class TTSNotifier extends Notifier<TTSState> {
  late final ConfigRepository _configRepo;

  @override
  TTSState build() {
    _configRepo = ref.watch(configRepoProvider);
    return const TTSState();
  }

  /// 加载 TTS 配置
  Future<void> load() async {
    try {
      final config = await _configRepo.getConfig();
      final tts = config.tts;

      final services = <TTSService>[
        const TTSService(
          id: 'kokoro',
          name: 'Kokoro 多语 v1.1',
          category: 'local',
          icon: CupertinoIcons.music_note_2,
          color: Color(0xFFAF52DE),
          description: '本地多语言 TTS 模型，音质自然',
        ),
        const TTSService(
          id: 'vits',
          name: 'VITS 中文 AIShell3',
          category: 'local',
          icon: CupertinoIcons.music_note_2,
          color: Color(0xFF5AC8FA),
          description: '中文语音合成模型',
          available: true,
        ),
        const TTSService(
          id: 'zipvoice',
          name: 'ZipVoice 声音克隆',
          category: 'local',
          icon: CupertinoIcons.person_2,
          color: Color(0xFFFF9500),
          description: '高质量声音克隆',
        ),
        const TTSService(
          id: 'openai_tts',
          name: 'OpenAI',
          category: 'online',
          icon: CupertinoIcons.sparkles,
          color: Color(0xFF10A37F),
          description: 'OpenAI TTS API',
        ),
        const TTSService(
          id: 'siliconflow',
          name: 'SiliconFlow 硅基流动',
          category: 'online',
          icon: CupertinoIcons.bolt,
          color: Color(0xFF007AFF),
          description: '国内 AI 语音服务',
        ),
        TTSService(
          id: 'local_tts',
          name: 'local',
          category: 'online',
          icon: CupertinoIcons.desktopcomputer,
          color: const Color(0xFF34C759),
          description: '本地 OpenAI 兼容 TTS 服务',
          available: tts.baseUrl.isNotEmpty,
        ),
        TTSService(
          id: 'edge_tts',
          name: 'Edge TTS',
          category: 'online',
          icon: CupertinoIcons.globe,
          color: const Color(0xFF5856D6),
          description: '微软 Edge 免费 TTS 服务，逆向接口，随时可能不可用',
          available: tts.type == 'edge',
          badge: 'FREE',
        ),
        const TTSService(
          id: 'azure',
          name: 'Azure',
          category: 'online',
          icon: CupertinoIcons.cloud,
          color: Color(0xFF007AFF),
          description: '微软 Azure 语音服务，企业级稳定',
        ),
        const TTSService(
          id: 'doubao',
          name: '豆包语音',
          category: 'online',
          icon: CupertinoIcons.music_note,
          color: Color(0xFFFF6B6B),
          description: '字节跳动豆包语音服务',
        ),
        const TTSService(
          id: 'elevenlabs',
          name: 'ElevenLabs',
          category: 'online',
          icon: CupertinoIcons.waveform_path_ecg,
          color: Color(0xFFFF9500),
          description: '高质量英文语音合成',
        ),
      ];

      state = state.copyWith(
        services: services,
        currentType: tts.type,
        currentVoice: tts.voice,
        config: tts,
      );
    } catch (_) {}
  }

  /// 选中服务
  void selectService(String? id) {
    state = state.copyWith(selectedServiceId: id, clearSelection: id == null);
  }

  /// 测试连接
  Future<void> testConnection(String serviceId) async {
    await Future.delayed(const Duration(seconds: 1));
  }

  /// saveConfig 将音色页的合成服务配置写回后端。
  Future<void> saveConfig(TTSConfig config) async {
    state = state.copyWith(saving: true);
    try {
      await _configRepo.updateConfig({
        'tts': {
          'type': config.type,
          'voice': config.voice,
          'base_url': config.baseUrl,
          if (!config.apiKey.contains('****')) 'api_key': config.apiKey,
          'model': config.model,
        },
      });
      state = state.copyWith(config: config, saving: false);
    } catch (_) {
      state = state.copyWith(saving: false);
      rethrow;
    }
  }
}

final ttsProvider = NotifierProvider<TTSNotifier, TTSState>(TTSNotifier.new);
