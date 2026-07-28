import '../services/api_client.dart';

/// 词库仓库——管理词库和词条
class TermRepository {
  final ApiClient _api;

  TermRepository({required this._api});

  // ---- Glossary ----

  Future<List<Map<String, dynamic>>> listGlossaries() => _api.listGlossaries();

  Future<Map<String, dynamic>> createGlossary(String name) =>
      _api.createGlossary(name);

  Future<Map<String, dynamic>> updateGlossary(
    int id, {
    String? name,
    bool? enabled,
    int? priority,
  }) =>
      _api.updateGlossary(id, name: name, enabled: enabled, priority: priority);

  Future<void> deleteGlossary(int id) => _api.deleteGlossary(id);

  // ---- Term ----

  Future<List<Map<String, dynamic>>> listTerms({
    required int glossaryId,
    String query = '',
  }) => _api.listTerms(glossaryId: glossaryId, query: query);

  Future<Map<String, dynamic>> createTerm({
    required int glossaryId,
    required String text,
    String translation = '',
    String note = '',
  }) => _api.createTerm(
    glossaryId: glossaryId,
    text: text,
    translation: translation,
    note: note,
  );

  Future<Map<String, dynamic>> updateTerm({
    required int glossaryId,
    required int id,
    required String text,
    required String translation,
    String note = '',
  }) => _api.updateTerm(
    glossaryId: glossaryId,
    id: id,
    text: text,
    translation: translation,
    note: note,
  );

  Future<void> deleteTerm({required int glossaryId, required int id}) =>
      _api.deleteTerm(glossaryId: glossaryId, id: id);
}
