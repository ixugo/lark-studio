import 'dart:async';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// 管理 Go 后端进程的生命周期 + 存活监控
class BackendService extends ChangeNotifier {
  Process? _process;
  bool _online = false;
  Timer? _healthTimer;
  final String _baseUrl;

  BackendService({String baseUrl = 'http://localhost:9523'}) : _baseUrl = baseUrl;

  bool get online => _online;
  bool get processRunning => _process != null;

  /// 启动 Go 后端并开启健康检查
  Future<void> start({String? binaryPath}) async {
    final bin = binaryPath ?? _findBinary();
    if (bin != null && _process == null) {
      try {
        _process = await Process.start(bin, []);
        _process!.exitCode.then((_) {
          _process = null;
          _setOnline(false);
        });
      } catch (e) {
        debugPrint('backend start failed: $e');
      }
    }
    _startHealthCheck();
  }

  /// 仅连接已运行的后端（不启动进程）
  void connectOnly() => _startHealthCheck();

  void _startHealthCheck() {
    _healthTimer?.cancel();
    _healthTimer = Timer.periodic(const Duration(seconds: 3), (_) => _check());
    _check();
  }

  Future<void> _check() async {
    try {
      final resp = await http.get(Uri.parse('$_baseUrl/health')).timeout(const Duration(seconds: 2));
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

  /// 在常见位置查找 vdub 可执行文件
  String? _findBinary() {
    final candidates = [
      '${Directory.current.path}/vdub',
      '${Directory.current.path}/build/darwin_arm64/bin',
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
    _process?.kill(ProcessSignal.sigterm);
    _process = null;
    _setOnline(false);
  }

  @override
  void dispose() {
    stop();
    super.dispose();
  }
}
