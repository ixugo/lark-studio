import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../data/repositories/term_repository.dart';
import '../../../../providers.dart';

/// 词库页状态
class GlossaryState {
  final List<Map<String, dynamic>> glossaries;
  final int? selectedGlossaryId;
  final List<Map<String, dynamic>> terms;
  final bool loadingGlossaries;
  final bool loadingTerms;
  final String? error;
  final String searchQuery;

  const GlossaryState({
    this.glossaries = const [],
    this.selectedGlossaryId,
    this.terms = const [],
    this.loadingGlossaries = true,
    this.loadingTerms = false,
    this.error,
    this.searchQuery = '',
  });

  GlossaryState copyWith({
    List<Map<String, dynamic>>? glossaries,
    int? selectedGlossaryId,
    bool clearSelection = false,
    List<Map<String, dynamic>>? terms,
    bool? loadingGlossaries,
    bool? loadingTerms,
    String? error,
    bool clearError = false,
    String? searchQuery,
  }) {
    return GlossaryState(
      glossaries: glossaries ?? this.glossaries,
      selectedGlossaryId:
          clearSelection ? null : (selectedGlossaryId ?? this.selectedGlossaryId),
      terms: terms ?? this.terms,
      loadingGlossaries: loadingGlossaries ?? this.loadingGlossaries,
      loadingTerms: loadingTerms ?? this.loadingTerms,
      error: clearError ? null : (error ?? this.error),
      searchQuery: searchQuery ?? this.searchQuery,
    );
  }
}

/// 词库状态管理
class GlossaryNotifier extends Notifier<GlossaryState> {
  late final TermRepository _repo;

  @override
  GlossaryState build() {
    _repo = ref.watch(termRepoProvider);
    return const GlossaryState();
  }

  /// 加载词库列表
  Future<void> loadGlossaries() async {
    state = state.copyWith(loadingGlossaries: true, clearError: true);
    try {
      final list = await _repo.listGlossaries();
      state = state.copyWith(glossaries: list, loadingGlossaries: false);
      // 自动选中第一个
      if (state.selectedGlossaryId == null && list.isNotEmpty) {
        selectGlossary((list.first['id'] as num).toInt());
      }
    } catch (e) {
      state = state.copyWith(error: '$e', loadingGlossaries: false);
    }
  }

  /// 选中词库 → 加载词条
  void selectGlossary(int id) {
    state = state.copyWith(selectedGlossaryId: id, searchQuery: '');
    loadTerms();
  }

  /// 加载当前词库的词条
  Future<void> loadTerms() async {
    final gid = state.selectedGlossaryId;
    if (gid == null) return;
    state = state.copyWith(loadingTerms: true);
    try {
      final terms =
          await _repo.listTerms(glossaryId: gid, query: state.searchQuery);
      state = state.copyWith(terms: terms, loadingTerms: false);
    } catch (e) {
      state = state.copyWith(error: '$e', loadingTerms: false);
    }
  }

  /// 搜索词条
  Future<void> search(String query) async {
    state = state.copyWith(searchQuery: query);
    await loadTerms();
  }

  /// 创建词库
  Future<void> createGlossary(String name) async {
    final result = await _repo.createGlossary(name);
    await loadGlossaries();
    final id = (result['id'] as num?)?.toInt();
    if (id != null) selectGlossary(id);
  }

  /// 更新词库
  Future<void> updateGlossary(int id,
      {String? name, bool? enabled, int? priority}) async {
    await _repo.updateGlossary(id,
        name: name, enabled: enabled, priority: priority);
    await loadGlossaries();
  }

  /// 删除词库
  Future<void> deleteGlossary(int id) async {
    await _repo.deleteGlossary(id);
    if (state.selectedGlossaryId == id) {
      state = state.copyWith(clearSelection: true, terms: []);
    }
    await loadGlossaries();
  }

  /// 添加词条
  Future<void> addTerm(String text,
      {String translation = '', String note = ''}) async {
    final gid = state.selectedGlossaryId;
    if (gid == null) return;
    await _repo.createTerm(
        glossaryId: gid, text: text, translation: translation, note: note);
    await loadTerms();
    await loadGlossaries();
  }

  /// 删除词条
  Future<void> deleteTerm(int id) async {
    final gid = state.selectedGlossaryId;
    if (gid == null) return;
    await _repo.deleteTerm(glossaryId: gid, id: id);
    await loadTerms();
    await loadGlossaries();
  }
}

final glossaryProvider =
    NotifierProvider<GlossaryNotifier, GlossaryState>(GlossaryNotifier.new);
