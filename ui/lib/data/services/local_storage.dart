import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

/// Flutter 本地持久化服务——基于 SharedPreferences 的 Map 存储
/// 用于存储工作流配置等轻量数据，无需依赖 Go 后端
class LocalStorage {
  static const _prefix = 'vdub_';
  SharedPreferences? _prefs;

  Future<SharedPreferences> _getPrefs() async {
    _prefs ??= await SharedPreferences.getInstance();
    return _prefs!;
  }

  Future<Map<String, dynamic>> getMap(String key) async {
    final prefs = await _getPrefs();
    final raw = prefs.getString('$_prefix$key');
    if (raw == null) return {};
    return jsonDecode(raw) as Map<String, dynamic>;
  }

  Future<void> setMap(String key, Map<String, dynamic> value) async {
    final prefs = await _getPrefs();
    await prefs.setString('$_prefix$key', jsonEncode(value));
  }

  Future<List<Map<String, dynamic>>> getList(String key) async {
    final prefs = await _getPrefs();
    final raw = prefs.getString('$_prefix$key');
    if (raw == null) return [];
    final list = jsonDecode(raw) as List<dynamic>;
    return list.cast<Map<String, dynamic>>();
  }

  Future<void> setList(
      String key, List<Map<String, dynamic>> value) async {
    final prefs = await _getPrefs();
    await prefs.setString('$_prefix$key', jsonEncode(value));
  }

  Future<void> remove(String key) async {
    final prefs = await _getPrefs();
    await prefs.remove('$_prefix$key');
  }
}
