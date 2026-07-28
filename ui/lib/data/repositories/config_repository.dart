import '../models/config.dart';
import '../services/api_client.dart';

/// 配置数据仓库——读取和更新应用配置，管理 Whisper 模型列表
class ConfigRepository {
  final ApiClient _api;
  AppConfig? _cached;

  ConfigRepository({required ApiClient api}) : _api = api;

  Future<AppConfig> getConfig({bool forceRefresh = false}) async {
    if (_cached != null && !forceRefresh) return _cached!;
    _cached = await _api.getConfig();
    return _cached!;
  }

  Future<AppConfig> updateConfig(Map<String, dynamic> updates) async {
    _cached = await _api.updateConfig(updates);
    return _cached!;
  }

  Future<List<Map<String, dynamic>>> listModels() => _api.listModels();

  Future<void> downloadModel(String name) => _api.downloadModel(name);

  /// 查询 whisper.cpp 运行时状态。
  Future<Map<String, dynamic>> getWhisperRuntime() => _api.getWhisperRuntime();

  /// 安装 whisper.cpp 运行时。
  Future<void> installWhisperRuntime() => _api.installWhisperRuntime();

  void invalidateCache() => _cached = null;
}
