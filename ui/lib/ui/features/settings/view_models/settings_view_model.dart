import 'dart:async';

import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/config.dart';
import '../../../../data/repositories/config_repository.dart';
import '../../../../data/services/websocket_service.dart';
import '../../../../providers.dart';

/// 设置页状态
class SettingsState {
  final AppConfig? config;
  final bool loading;
  final bool saving;
  final String? error;
  final List<Map<String, dynamic>> models;

  const SettingsState({
    this.config,
    this.loading = true,
    this.saving = false,
    this.error,
    this.models = const [],
  });

  SettingsState copyWith({
    AppConfig? config,
    bool? loading,
    bool? saving,
    String? error,
    bool clearError = false,
    List<Map<String, dynamic>>? models,
  }) {
    return SettingsState(
      config: config ?? this.config,
      loading: loading ?? this.loading,
      saving: saving ?? this.saving,
      error: clearError ? null : (error ?? this.error),
      models: models ?? this.models,
    );
  }
}

/// 设置页状态管理
class SettingsNotifier extends Notifier<SettingsState> {
  late final ConfigRepository _configRepo;

  @override
  SettingsState build() {
    _configRepo = ref.watch(configRepoProvider);
    final subscription = ref
        .watch(wsServiceProvider)
        .events
        .listen(_handleModelEvent);
    ref.onDispose(subscription.cancel);
    return const SettingsState();
  }

  /// 处理模型下载事件，使下载百分比无需轮询即可更新。
  void _handleModelEvent(WsEvent event) {
    if (!event.type.startsWith('model_download_')) return;
    final name = event.data['model'] as String? ?? '';
    if (name.isEmpty) return;
    if (event.type == 'model_download_progress') {
      final progress = event.data['progress'] as int? ?? 0;
      state = state.copyWith(
        models: state.models
            .map(
              (model) => model['name'] == name
                  ? {...model, 'downloading': true, 'progress': progress}
                  : model,
            )
            .toList(),
      );
      return;
    }
    unawaited(_refreshModels());
  }

  /// 刷新模型磁盘状态，供下载完成和失败事件复用。
  Future<void> _refreshModels() async {
    try {
      state = state.copyWith(models: await _configRepo.listModels());
    } catch (_) {}
  }

  Future<void> loadConfig() async {
    state = state.copyWith(loading: true, clearError: true);
    try {
      final cfg = await _configRepo.getConfig(forceRefresh: true);
      var models = <Map<String, dynamic>>[];
      try {
        models = await _configRepo.listModels();
      } catch (_) {}
      state = state.copyWith(config: cfg, models: models, loading: false);
    } catch (e) {
      state = state.copyWith(error: '$e', loading: false);
    }
  }

  Future<AppConfig?> saveConfig(Map<String, dynamic> updates) async {
    state = state.copyWith(saving: true);
    try {
      final cfg = await _configRepo.updateConfig(updates);
      state = state.copyWith(config: cfg, saving: false);
      return cfg;
    } catch (e) {
      state = state.copyWith(saving: false);
      rethrow;
    }
  }

  Future<void> downloadModel(String name) async {
    await _configRepo.downloadModel(name);
    state = state.copyWith(
      models: state.models
          .map(
            (model) => model['name'] == name
                ? {...model, 'downloading': true, 'progress': 0}
                : model,
          )
          .toList(),
    );
  }
}

final settingsProvider = NotifierProvider<SettingsNotifier, SettingsState>(
  SettingsNotifier.new,
);
