import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app.dart';
import 'services/api_client.dart';
import 'services/backend_service.dart';
import 'services/websocket_service.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  final backend = BackendService();
  final api = ApiClient(baseUrl: 'http://localhost:9523');
  final wsService = WebSocketService();
  var configSynced = false;

  backend.addListener(() {
    if (backend.port > 0) {
      api.updateBaseUrl('http://localhost:${backend.port}');
      wsService.updateWsUrl('ws://localhost:${backend.port}/ws');
    }
    if (backend.online && !wsService.connected) {
      wsService.connect();
    }
    if (backend.online && !configSynced) {
      configSynced = true;
      _syncOnboardingConfig(api);
    }
  });
  backend.start();

  runApp(
    MultiProvider(
      providers: [
        ChangeNotifierProvider<BackendService>.value(value: backend),
        Provider<ApiClient>.value(value: api),
        ChangeNotifierProvider<WebSocketService>.value(value: wsService),
        ChangeNotifierProvider(create: (_) => TaskListNotifier(api)),
      ],
      child: const VdubApp(),
    ),
  );
}

/// 引擎首次上线时，将 onboarding 保存的 whisper_mode 同步到 Go 后端
Future<void> _syncOnboardingConfig(ApiClient api) async {
  try {
    final prefs = await SharedPreferences.getInstance();
    final mode = prefs.getString('whisper_mode');
    if (mode == null) return;
    await api.updateConfig({
      'pipeline': {'whisper_mode': mode},
    });
  } catch (e) {
    debugPrint('sync onboarding config failed: $e');
  }
}
