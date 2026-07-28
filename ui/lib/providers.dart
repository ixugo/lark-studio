import 'dart:ui' show Brightness;
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'data/repositories/config_repository.dart';
import 'data/repositories/task_repository.dart';
import 'data/repositories/term_repository.dart';
import 'data/services/api_client.dart';
import 'data/services/backend_service.dart';
import 'data/services/local_storage.dart';
import 'data/services/websocket_service.dart';

// ---- Services ----
// BackendService/WebSocketService 是 ChangeNotifier，用 Provider 持有单例
// View 层通过 useListenable(ref.watch(xxxProvider)) 监听变化

final backendProvider = Provider<BackendService>((ref) {
  final svc = BackendService();
  ref.onDispose(() => svc.dispose());
  return svc;
});

final apiClientProvider = Provider<ApiClient>(
  (ref) => ApiClient(baseUrl: 'http://127.0.0.1:9523'),
);

final wsServiceProvider = Provider<WebSocketService>((ref) {
  final svc = WebSocketService();
  ref.onDispose(() => svc.dispose());
  return svc;
});

final localStorageProvider = Provider<LocalStorage>((ref) => LocalStorage());

// ---- Repositories ----

final taskRepoProvider = Provider<TaskRepository>((ref) {
  return TaskRepository(api: ref.watch(apiClientProvider));
});

final configRepoProvider = Provider<ConfigRepository>((ref) {
  return ConfigRepository(api: ref.watch(apiClientProvider));
});

final termRepoProvider = Provider<TermRepository>((ref) {
  return TermRepository(api: ref.watch(apiClientProvider));
});

// ---- Theme ----

/// 主题模式：system / light / dark
enum AppThemeMode { system, light, dark }

class ThemeNotifier extends Notifier<AppThemeMode> {
  static const _key = 'vdub_theme_mode';

  @override
  AppThemeMode build() {
    _load();
    return AppThemeMode.system;
  }

  Future<void> _load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_key);
    if (raw == null) return;
    final mode = AppThemeMode.values.firstWhere(
      (e) => e.name == raw,
      orElse: () => AppThemeMode.system,
    );
    state = mode;
  }

  /// 在 system / light / dark 间循环
  Future<void> toggle() async {
    final next = switch (state) {
      AppThemeMode.system => AppThemeMode.dark,
      AppThemeMode.dark => AppThemeMode.light,
      AppThemeMode.light => AppThemeMode.system,
    };
    state = next;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_key, next.name);
  }

  /// 解析为实际 Brightness
  Brightness resolve(Brightness platformBrightness) {
    return switch (state) {
      AppThemeMode.light => Brightness.light,
      AppThemeMode.dark => Brightness.dark,
      AppThemeMode.system => platformBrightness,
    };
  }
}

final themeProvider = NotifierProvider<ThemeNotifier, AppThemeMode>(
  ThemeNotifier.new,
);
