import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import 'app.dart';
import 'services/api_client.dart';
import 'services/backend_service.dart';
import 'services/websocket_service.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();

  final backend = BackendService();
  final api = ApiClient(baseUrl: 'http://localhost:9523');
  final wsService = WebSocketService();

  backend.addListener(() {
    if (backend.port > 0) {
      api.updateBaseUrl('http://localhost:${backend.port}');
      wsService.updateWsUrl('ws://localhost:${backend.port}/ws');
    }
    if (backend.online && !wsService.connected) {
      wsService.connect();
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
