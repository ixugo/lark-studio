import 'dart:async';
import 'dart:io';
import 'dart:math';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// 管理 Go 引擎进程的生命周期 + 存活监控
/// 启动时自动选择可用端口，关闭时杀死引擎进程
class BackendService extends ChangeNotifier {
  Process? _process;
  bool _online = false;
  Timer? _healthTimer;
  int _port = 0;

  BackendService();

  bool get online => _online;
  bool get processRunning => _process != null;
  int get port => _port;
  String get baseUrl => 'http://localhost:$_port';
  String get wsUrl => 'ws://localhost:$_port/ws';

  /// 启动引擎：找空闲端口 → 拉起子进程 → 健康检查
  /// 若二进制未找到，回退到 connectOnly 模式（开发者可手动启动引擎）
  Future<bool> start({String? binaryPath}) async {
    final bin = binaryPath ?? _findBinary();
    if (bin == null) {
      debugPrint('engine binary not found, falling back to connectOnly');
      connectOnly();
      return false;
    }

    _port = await _findFreePort();
    try {
      _process = await Process.start(bin, ['-port', '$_port']);
      _process!.stdout.listen((data) {
        debugPrint('engine: ${String.fromCharCodes(data).trim()}');
      });
      _process!.stderr.listen((data) {
        debugPrint('engine err: ${String.fromCharCodes(data).trim()}');
      });
      _process!.exitCode.then((_) {
        _process = null;
        _setOnline(false);
      });
    } catch (e) {
      debugPrint('engine start failed: $e');
      return false;
    }
    _startHealthCheck();
    return true;
  }

  /// 仅连接已运行的引擎（调试时用 -port 手动启动）
  void connectOnly({int port = 9523}) {
    _port = port;
    _startHealthCheck();
  }

  void _startHealthCheck() {
    _healthTimer?.cancel();
    _healthTimer = Timer.periodic(const Duration(seconds: 3), (_) => _check());
    _check();
  }

  Future<void> _check() async {
    try {
      final resp = await http.get(Uri.parse('$baseUrl/health')).timeout(const Duration(seconds: 2));
      _setOnline(resp.statusCode == 200);
    } catch (_) {
      _setOnline(false);
    }
  }

  void _setOnline(bool v) {
    if (_online != v) {
      _online = v;
      notifyListeners();
    }
  }

  /// 在 9523~9623 范围内找一个可用端口
  Future<int> _findFreePort() async {
    final rng = Random();
    for (var i = 0; i < 20; i++) {
      final candidate = 9523 + rng.nextInt(100);
      try {
        final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, candidate);
        await socket.close();
        return candidate;
      } catch (_) {
        continue;
      }
    }
    // fallback: 让 OS 分配
    final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final port = socket.port;
    await socket.close();
    return port;
  }

  /// 在常见位置查找 vdub 可执行文件
  String? _findBinary() {
    final candidates = [
      '${Directory.current.path}/vdub',
      '${Directory.current.path}/build/darwin_arm64/bin',
      '${Directory.current.path}/../vdub',
      '${Directory.current.path}/../build/darwin_arm64/bin',
      '${Directory.current.path}/../tmp/vdub',
    ];
    for (final path in candidates) {
      if (File(path).existsSync()) return path;
    }
    return null;
  }

  void stop() {
    _healthTimer?.cancel();
    if (_process != null) {
      _process!.kill(ProcessSignal.sigterm);
      _process = null;
    }
    _setOnline(false);
  }

  @override
  void dispose() {
    stop();
    super.dispose();
  }
}
