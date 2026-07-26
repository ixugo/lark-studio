import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// WebSocket 客户端：心跳保活 + 接收任务事件
class WebSocketService extends ChangeNotifier {
  String _wsUrl;
  WebSocketChannel? _channel;
  Timer? _heartbeatTimer;
  Timer? _reconnectTimer;
  bool _connected = false;

  final _eventController = StreamController<WsEvent>.broadcast();

  // ignore: prefer_initializing_formals
  WebSocketService({String wsUrl = 'ws://localhost:9523/ws'}) : _wsUrl = wsUrl;

  bool get connected => _connected;
  Stream<WsEvent> get events => _eventController.stream;

  void updateWsUrl(String url) => _wsUrl = url;

  void connect() {
    _doConnect();
  }

  void _doConnect() {
    try {
      _channel?.sink.close();
      _channel = WebSocketChannel.connect(Uri.parse(_wsUrl));
      _connected = true;
      notifyListeners();

      _startHeartbeat();

      _channel!.stream.listen(
        _onMessage,
        onError: (e) {
          debugPrint('ws error: $e');
          _onDisconnect();
        },
        onDone: _onDisconnect,
      );
    } catch (e) {
      debugPrint('ws connect failed: $e');
      _scheduleReconnect();
    }
  }

  void _onMessage(dynamic data) {
    try {
      final json = jsonDecode(data as String) as Map<String, dynamic>;
      final type = json['type'] as String? ?? '';
      final payload = json['data'] as Map<String, dynamic>? ?? {};
      _eventController.add(WsEvent(type, payload));
    } catch (e) {
      debugPrint('ws parse error: $e');
    }
  }

  void _startHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(const Duration(seconds: 25), (_) {
      _send('heartbeat', {});
    });
  }

  void _send(String type, Map<String, dynamic> data) {
    try {
      _channel?.sink.add(jsonEncode({'type': type, 'data': data}));
    } catch (_) {}
  }

  void _onDisconnect() {
    _connected = false;
    _heartbeatTimer?.cancel();
    notifyListeners();
    _scheduleReconnect();
  }

  void _scheduleReconnect() {
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(const Duration(seconds: 3), _doConnect);
  }

  @override
  void dispose() {
    _heartbeatTimer?.cancel();
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    _eventController.close();
    super.dispose();
  }
}

class WsEvent {
  final String type;
  final Map<String, dynamic> data;
  const WsEvent(this.type, this.data);
}
