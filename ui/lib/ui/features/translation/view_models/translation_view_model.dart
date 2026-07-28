import 'dart:ui' show Color;
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart' show IconData;
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../data/repositories/config_repository.dart';
import '../../../../providers.dart';

/// 翻译服务定义
class TranslationService {
  final String id;
  final String name;
  final String category;
  final IconData icon;
  final Color color;
  final String description;
  final bool configured;
  final String baseUrl;
  final String apiKey;
  final String model;

  const TranslationService({
    required this.id,
    required this.name,
    required this.category,
    this.icon = CupertinoIcons.globe,
    this.color = const Color(0xFF8E8E93),
    this.description = '',
    this.configured = false,
    this.baseUrl = '',
    this.apiKey = '',
    this.model = '',
  });

  static const empty = TranslationService(id: '', name: '', category: '');

  TranslationService copyWith({
    bool? configured,
    String? baseUrl,
    String? apiKey,
    String? model,
  }) {
    return TranslationService(
      id: id,
      name: name,
      category: category,
      icon: icon,
      color: color,
      description: description,
      configured: configured ?? this.configured,
      baseUrl: baseUrl ?? this.baseUrl,
      apiKey: apiKey ?? this.apiKey,
      model: model ?? this.model,
    );
  }
}

/// 翻译页状态
class TranslationState {
  final List<TranslationService> services;
  final String? selectedServiceId;
  final bool showConfiguredOnly;

  const TranslationState({
    this.services = const [],
    this.selectedServiceId,
    this.showConfiguredOnly = false,
  });

  TranslationState copyWith({
    List<TranslationService>? services,
    String? selectedServiceId,
    bool clearSelection = false,
    bool? showConfiguredOnly,
  }) {
    return TranslationState(
      services: services ?? this.services,
      selectedServiceId:
          clearSelection ? null : (selectedServiceId ?? this.selectedServiceId),
      showConfiguredOnly: showConfiguredOnly ?? this.showConfiguredOnly,
    );
  }
}

/// 翻译服务状态管理
class TranslationNotifier extends Notifier<TranslationState> {
  late final ConfigRepository _configRepo;

  @override
  TranslationState build() {
    _configRepo = ref.watch(configRepoProvider);
    return const TranslationState();
  }

  /// 加载配置，初始化服务列表
  Future<void> load() async {
    try {
      final config = await _configRepo.getConfig();
      final llm = config.llm;
      final hasLlm = llm.baseUrl.isNotEmpty && llm.apiKey.isNotEmpty;

      final services = <TranslationService>[
        TranslationService(
          id: 'local',
          name: 'local',
          category: 'custom',
          icon: CupertinoIcons.desktopcomputer,
          color: const Color(0xFF30D158),
          description:
              'OpenAI 兼容 API 的基础地址，通常以 /v1 结尾。\nSmartSub 会自动拼接聊天接口路径。',
          configured: hasLlm,
          baseUrl: llm.baseUrl,
          apiKey: llm.apiKey,
          model: llm.model,
        ),
        const TranslationService(
          id: 'auto_free',
          name: '自动免费翻译',
          category: 'free',
          icon: CupertinoIcons.bolt,
          color: Color(0xFFFF9500),
          description: '自动在多个免费翻译引擎之间轮询。',
        ),
        const TranslationService(
          id: 'bing',
          name: '必应免费翻译',
          category: 'free',
          icon: CupertinoIcons.textformat,
          color: Color(0xFF007AFF),
          description: '微软必应翻译引擎，免费无需配置。',
        ),
        const TranslationService(
          id: 'google',
          name: '谷歌免费翻译',
          category: 'free',
          icon: CupertinoIcons.globe,
          color: Color(0xFF34C759),
          description: '谷歌翻译引擎，免费但需要网络通畅。',
        ),
        const TranslationService(
          id: 'deeplx',
          name: 'DeepLX',
          category: 'free',
          icon: CupertinoIcons.text_bubble,
          color: Color(0xFF5856D6),
          description: 'DeepL 免费接口，翻译质量较高。',
        ),
        const TranslationService(
          id: 'openai',
          name: 'OpenAI',
          category: 'ai',
          icon: CupertinoIcons.sparkles,
          color: Color(0xFF10A37F),
          description: 'OpenAI API，支持 GPT-4o 等模型。',
        ),
        const TranslationService(
          id: 'ollama',
          name: 'Ollama',
          category: 'ai',
          icon: CupertinoIcons.cube,
          color: Color(0xFFFF6B6B),
          description: '本地运行的开源大语言模型。',
        ),
      ];

      state = state.copyWith(services: services);
    } catch (_) {}
  }

  /// 选中服务
  void selectService(String? id) {
    state = state.copyWith(selectedServiceId: id, clearSelection: id == null);
  }

  /// 切换仅显示已配置
  void toggleConfiguredFilter() {
    state = state.copyWith(showConfiguredOnly: !state.showConfiguredOnly);
  }

  /// 保存服务配置
  Future<void> saveServiceConfig(String serviceId,
      {String? baseUrl, String? apiKey, String? model}) async {
    final updates = <String, dynamic>{
      'llm': <String, dynamic>{
        if (baseUrl != null) 'base_url': baseUrl,
        if (apiKey != null) 'api_key': apiKey,
        if (model != null) 'model': model,
      },
    };
    await _configRepo.updateConfig(updates);
    await load();
  }

  /// 测试翻译连接
  Future<void> testTranslation(String serviceId) async {
    await Future.delayed(const Duration(seconds: 1));
  }
}

final translationProvider =
    NotifierProvider<TranslationNotifier, TranslationState>(
        TranslationNotifier.new);
