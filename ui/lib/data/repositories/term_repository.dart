import '../services/api_client.dart';

/// 术语词库仓库——管理翻译术语锁定
class TermRepository {
  final ApiClient _api;

  TermRepository({required ApiClient api}) : _api = api;

  Future<List<Map<String, dynamic>>> listTerms() => _api.listTerms();

  Future<Map<String, dynamic>> createTerm(String text,
      {String translation = ''}) =>
      _api.createTerm(text, translation: translation);

  Future<void> deleteTerm(int id) => _api.deleteTerm(id);
}
