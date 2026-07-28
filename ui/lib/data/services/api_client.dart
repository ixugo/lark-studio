import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/config.dart';
import '../models/task.dart';

/// Go 引擎 HTTP 客户端——纯粹的无状态 Service 层，只负责发请求和解析响应
class ApiClient {
  String baseUrl;
  final http.Client _client;

  ApiClient({required this.baseUrl}) : _client = http.Client();

  void updateBaseUrl(String url) => baseUrl = url;

  Future<bool> healthCheck() async {
    try {
      final resp = await _client
          .get(Uri.parse('$baseUrl/health'))
          .timeout(const Duration(seconds: 2));
      return resp.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  // ---- Task API ----

  Future<List<Task>> listTasks({int page = 1, int size = 50}) async {
    final uri = Uri.parse('$baseUrl/tasks').replace(
      queryParameters: {'page': '$page', 'size': '$size'},
    );
    final resp = await _client.get(uri);
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>? ?? {};
    final items = data['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => Task.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<Task> getTask(String id) async {
    final resp = await _client.get(Uri.parse('$baseUrl/tasks/$id'));
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return Task.fromJson(body['data'] as Map<String, dynamic>? ?? {});
  }

  Future<Task> createTask({
    required String inputPath,
    required int mode,
    String outputDir = '',
    String targetLang = '',
  }) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/tasks'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'input_path': inputPath,
        'mode': mode,
        if (outputDir.isNotEmpty) 'output_dir': outputDir,
        if (targetLang.isNotEmpty) 'target_lang': targetLang,
      }),
    );
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return Task.fromJson(body['data'] as Map<String, dynamic>? ?? {});
  }

  Future<void> deleteTask(String id) async {
    final resp = await _client.delete(Uri.parse('$baseUrl/tasks/$id'));
    _checkStatus(resp);
  }

  Future<void> pauseTask(String id) async {
    final resp = await _client.post(Uri.parse('$baseUrl/tasks/$id/pause'));
    _checkStatus(resp);
  }

  Future<Task> resumeTask(String id) async {
    final resp = await _client.post(Uri.parse('$baseUrl/tasks/$id/resume'));
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return Task.fromJson(body['data'] as Map<String, dynamic>? ?? {});
  }

  Future<List<Task>> batchCreateTasks({
    required List<String> videos,
    required int mode,
    String targetLang = '',
  }) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/tasks/batch'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'videos': videos,
        'mode': mode,
        if (targetLang.isNotEmpty) 'target_lang': targetLang,
      }),
    );
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>? ?? {};
    final items = data['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => Task.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  // ---- Glossary API ----

  Future<List<Map<String, dynamic>>> listGlossaries() async {
    final resp = await _client.get(Uri.parse('$baseUrl/glossaries'));
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>? ?? {};
    final items = data['items'] as List<dynamic>? ?? [];
    return items.cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> createGlossary(String name) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/glossaries'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'name': name}),
    );
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return body['data'] as Map<String, dynamic>? ?? {};
  }

  Future<Map<String, dynamic>> updateGlossary(int id, {String? name, bool? enabled, int? priority}) async {
    final updates = <String, dynamic>{};
    if (name != null) updates['name'] = name;
    if (enabled != null) updates['enabled'] = enabled;
    if (priority != null) updates['priority'] = priority;
    final resp = await _client.put(
      Uri.parse('$baseUrl/glossaries/$id'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(updates),
    );
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return body['data'] as Map<String, dynamic>? ?? {};
  }

  Future<void> deleteGlossary(int id) async {
    final resp = await _client.delete(Uri.parse('$baseUrl/glossaries/$id'));
    _checkStatus(resp);
  }

  // ---- Term API (per glossary) ----

  Future<List<Map<String, dynamic>>> listTerms({required int glossaryId, String query = ''}) async {
    var url = '$baseUrl/glossaries/$glossaryId/terms';
    if (query.isNotEmpty) url += '?q=${Uri.encodeComponent(query)}';
    final resp = await _client.get(Uri.parse(url));
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>? ?? {};
    final items = data['items'] as List<dynamic>? ?? [];
    return items.cast<Map<String, dynamic>>();
  }

  Future<Map<String, dynamic>> createTerm({
    required int glossaryId,
    required String text,
    String translation = '',
    String note = '',
  }) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/glossaries/$glossaryId/terms'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'text': text,
        if (translation.isNotEmpty) 'translation': translation,
        if (note.isNotEmpty) 'note': note,
      }),
    );
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return body['data'] as Map<String, dynamic>? ?? {};
  }

  Future<void> deleteTerm({required int glossaryId, required int id}) async {
    final resp = await _client.delete(Uri.parse('$baseUrl/glossaries/$glossaryId/terms/$id'));
    _checkStatus(resp);
  }

  // ---- Model API ----

  Future<List<Map<String, dynamic>>> listModels() async {
    final resp = await _client.get(Uri.parse('$baseUrl/models'));
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    final data = body['data'] as List<dynamic>? ?? [];
    return data.cast<Map<String, dynamic>>();
  }

  Future<void> downloadModel(String name) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/models/download'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'name': name}),
    );
    _checkStatus(resp);
  }

  // ---- Config API ----

  Future<AppConfig> getConfig() async {
    final resp = await _client.get(Uri.parse('$baseUrl/config'));
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return AppConfig.fromJson(body['data'] as Map<String, dynamic>? ?? {});
  }

  Future<AppConfig> updateConfig(Map<String, dynamic> updates) async {
    final resp = await _client.put(
      Uri.parse('$baseUrl/config'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode(updates),
    );
    _checkStatus(resp);
    final body = jsonDecode(resp.body) as Map<String, dynamic>;
    return AppConfig.fromJson(body['data'] as Map<String, dynamic>? ?? {});
  }

  void _checkStatus(http.Response resp) {
    if (resp.statusCode >= 400) {
      final body = jsonDecode(resp.body) as Map<String, dynamic>?;
      final msg = body?['message'] as String? ?? 'request failed';
      throw ApiException(resp.statusCode, msg);
    }
  }

  void dispose() => _client.close();
}

class ApiException implements Exception {
  final int statusCode;
  final String message;
  const ApiException(this.statusCode, this.message);
  @override
  String toString() => 'ApiException($statusCode): $message';
}
