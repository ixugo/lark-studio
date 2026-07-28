import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

/// 管理 Go 引擎进程的生命周期 + 存活监控
/// 启动时由引擎原子分配回环端口，关闭时杀死引擎进程
/// 健康检查连续失败 3 次后自动重启引擎
class BackendService extends ChangeNotifier {
  Process? _process;
  bool _online = false;
  Timer? _healthTimer;
  int _port = 0;
  String? _binaryPath;
  int _failCount = 0;
  bool _restarting = false;

  static const _maxFailBeforeRestart = 3;

  BackendService();

  bool get online => _online;
  bool get processRunning => _process != null;
  int get port => _port;
  String get baseUrl => 'http://127.0.0.1:$_port';
  String get wsUrl => 'ws://127.0.0.1:$_port/ws';

  /// 启动引擎：拉起子进程 → 接收实际端口 → 健康检查
  /// 若二进制未找到，回退到 connectOnly 模式（开发者可手动启动引擎）
  Future<bool> start({String? binaryPath}) async {
    final bin = binaryPath ?? _findBinary();
    if (bin == null) {
      debugPrint('engine binary not found, falling back to connectOnly');
      connectOnly();
      return false;
    }
    _binaryPath = bin;
    return _launchProcess(bin);
  }

  Future<bool> _launchProcess(String bin) async {
    _port = 0;
    try {
      _process = await Process.start(bin, const ['-port', '0']);
      _process!.stdout
          .transform(utf8.decoder)
          .transform(const LineSplitter())
          .listen((line) {
            _readEngineLine(line);
          });
      _process!.stderr.listen((data) {
        debugPrint('engine err: ${String.fromCharCodes(data).trim()}');
      });
      _process!.exitCode.then((_) {
        _process = null;
        _port = 0;
        _setOnline(false);
      });
    } catch (e) {
      debugPrint('engine start failed: $e');
      return false;
    }
    _failCount = 0;
    return true;
  }

  /// 解析引擎就绪行；端口由 Go 原子绑定，避免前端探测端口的竞态。
  void _readEngineLine(String line) {
    debugPrint('engine: $line');
    final match = RegExp(r'^ENGINE_PORT=(\d+)$').firstMatch(line.trim());
    if (match == null) return;
    _port = int.parse(match.group(1)!);
    notifyListeners();
    _startHealthCheck();
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
      final resp = await http
          .get(Uri.parse('$baseUrl/health'))
          .timeout(const Duration(seconds: 2));
      if (resp.statusCode == 200) {
        _failCount = 0;
        _setOnline(true);
        return;
      }
    } catch (_) {}

    _failCount++;
    _setOnline(false);

    if (_binaryPath != null &&
        _failCount >= _maxFailBeforeRestart &&
        !_restarting) {
      _restarting = true;
      debugPrint('engine health check failed $_failCount times, restarting...');
      _killProcess();
      await Future.delayed(const Duration(seconds: 1));
      await _launchProcess(_binaryPath!);
      _restarting = false;
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
    final exe = Platform.resolvedExecutable;
    final goos = _goOS();
    final arch = _arch();
    final candidates = <String>[];

    if (Platform.isMacOS && exe.contains('.app/Contents/')) {
      final contentsDir = exe.substring(
        0,
        exe.indexOf('.app/Contents/') + '.app/Contents'.length,
      );
      candidates.add('$contentsDir/Resources/vdub');

      final appDir = exe.substring(0, exe.indexOf('.app/'));
      final uiDir = appDir.contains('/ui/')
          ? appDir.substring(0, appDir.indexOf('/ui/'))
          : null;
      if (uiDir != null) {
        candidates.add('$uiDir/build/${goos}_$arch/vdub');
        candidates.add('$uiDir/vdub');
      }
    }

    candidates.addAll([
      '${Directory.current.path}/vdub',
      '${Directory.current.path}/build/${goos}_$arch/vdub',
      '${Directory.current.path}/../vdub',
      '${Directory.current.path}/../build/${goos}_$arch/vdub',
    ]);

    for (final path in candidates) {
      debugPrint('searching engine: $path');
      if (File(path).existsSync()) {
        debugPrint('engine found: $path');
        return path;
      }
    }
    return null;
  }

  static String _arch() {
    final dart = Platform.version;
    if (dart.contains('arm64') || dart.contains('aarch64')) return 'arm64';
    return 'amd64';
  }

  static String _goOS() {
    if (Platform.isMacOS) return 'darwin';
    if (Platform.isWindows) return 'windows';
    return 'linux';
  }

  void _killProcess() {
    if (_process != null) {
      _process!.kill(ProcessSignal.sigterm);
      _process = null;
    }
  }

  void stop() {
    _healthTimer?.cancel();
    _killProcess();
    _setOnline(false);
  }

  @override
  void dispose() {
    stop();
    super.dispose();
  }
}
