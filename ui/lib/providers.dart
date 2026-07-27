import 'package:flutter_riverpod/flutter_riverpod.dart';

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

final apiClientProvider =
    Provider<ApiClient>((ref) => ApiClient(baseUrl: 'http://localhost:9523'));

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
