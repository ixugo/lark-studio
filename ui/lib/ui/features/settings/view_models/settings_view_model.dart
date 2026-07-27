import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../data/models/config.dart';
import '../../../../data/repositories/config_repository.dart';
import '../../../../data/repositories/term_repository.dart';
import '../../../../providers.dart';

/// 设置页状态
class SettingsState {
  final AppConfig? config;
  final bool loading;
  final bool saving;
  final String? error;
  final List<Map<String, dynamic>> terms;
  final List<Map<String, dynamic>> models;

  const SettingsState({
    this.config,
    this.loading = true,
    this.saving = false,
    this.error,
    this.terms = const [],
    this.models = const [],
  });

  SettingsState copyWith({
    AppConfig? config,
    bool? loading,
    bool? saving,
    String? error,
    bool clearError = false,
    List<Map<String, dynamic>>? terms,
    List<Map<String, dynamic>>? models,
  }) {
    return SettingsState(
      config: config ?? this.config,
      loading: loading ?? this.loading,
      saving: saving ?? this.saving,
      error: clearError ? null : (error ?? this.error),
      terms: terms ?? this.terms,
      models: models ?? this.models,
    );
  }
}

/// 设置页状态管理
class SettingsNotifier extends Notifier<SettingsState> {
  late final ConfigRepository _configRepo;
  late final TermRepository _termRepo;

  @override
  SettingsState build() {
    _configRepo = ref.watch(configRepoProvider);
    _termRepo = ref.watch(termRepoProvider);
    return const SettingsState();
  }

  Future<void> loadConfig() async {
    state = state.copyWith(loading: true, clearError: true);
    try {
      final cfg = await _configRepo.getConfig(forceRefresh: true);
      var terms = <Map<String, dynamic>>[];
      try {
        terms = await _termRepo.listTerms();
      } catch (_) {}
      var models = <Map<String, dynamic>>[];
      try {
        models = await _configRepo.listModels();
      } catch (_) {}
      state = state.copyWith(
          config: cfg, terms: terms, models: models, loading: false);
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

  Future<void> addTerm(String text, {String translation = ''}) async {
    await _termRepo.createTerm(text, translation: translation);
    final terms = await _termRepo.listTerms();
    state = state.copyWith(terms: terms);
  }

  Future<void> deleteTerm(int id) async {
    await _termRepo.deleteTerm(id);
    final terms = await _termRepo.listTerms();
    state = state.copyWith(terms: terms);
  }

  Future<void> downloadModel(String name) async {
    await _configRepo.downloadModel(name);
    await Future.delayed(const Duration(seconds: 2));
    try {
      final models = await _configRepo.listModels();
      state = state.copyWith(models: models);
    } catch (_) {}
  }
}

final settingsProvider =
    NotifierProvider<SettingsNotifier, SettingsState>(SettingsNotifier.new);
