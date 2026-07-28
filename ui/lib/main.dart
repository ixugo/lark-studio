import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app.dart';
import 'data/services/api_client.dart';
import 'providers.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  runApp(ProviderScope(child: const _AppBootstrap()));
}

/// 读取 providers 启动引擎和 WebSocket
class _AppBootstrap extends ConsumerStatefulWidget {
  const _AppBootstrap();

  @override
  ConsumerState<_AppBootstrap> createState() => _AppBootstrapState();
}

class _AppBootstrapState extends ConsumerState<_AppBootstrap> {
  bool _configSynced = false;

  @override
  void initState() {
    super.initState();
    final backend = ref.read(backendProvider);
    final api = ref.read(apiClientProvider);
    final ws = ref.read(wsServiceProvider);

    backend.addListener(() {
      if (backend.port > 0) {
        api.updateBaseUrl('http://127.0.0.1:${backend.port}');
        ws.updateWsUrl('ws://127.0.0.1:${backend.port}/ws');
      }
      if (backend.online && !ws.connected) {
        ws.connect();
      }
      if (backend.online && !_configSynced) {
        _configSynced = true;
        _syncOnboardingConfig(api);
      }
    });
    backend.start();
  }

  @override
  Widget build(BuildContext context) => const VdubApp();
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
