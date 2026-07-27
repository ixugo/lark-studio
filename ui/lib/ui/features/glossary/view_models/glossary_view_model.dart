import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../data/repositories/term_repository.dart';
import '../../../../providers.dart';

/// 词库状态
class GlossaryState {
  final List<Map<String, dynamic>> terms;
  final bool loading;
  final String? error;

  const GlossaryState({
    this.terms = const [],
    this.loading = true,
    this.error,
  });

  GlossaryState copyWith({
    List<Map<String, dynamic>>? terms,
    bool? loading,
    String? error,
    bool clearError = false,
  }) {
    return GlossaryState(
      terms: terms ?? this.terms,
      loading: loading ?? this.loading,
      error: clearError ? null : (error ?? this.error),
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

  Future<void> load() async {
    state = state.copyWith(loading: true, clearError: true);
    try {
      final terms = await _repo.listTerms();
      state = state.copyWith(terms: terms, loading: false);
    } catch (e) {
      state = state.copyWith(error: '$e', loading: false);
    }
  }

  Future<void> addTerm(String text, {String translation = ''}) async {
    await _repo.createTerm(text, translation: translation);
    final terms = await _repo.listTerms();
    state = state.copyWith(terms: terms);
  }

  Future<void> deleteTerm(int id) async {
    await _repo.deleteTerm(id);
    final terms = await _repo.listTerms();
    state = state.copyWith(terms: terms);
  }
}

final glossaryProvider =
    NotifierProvider<GlossaryNotifier, GlossaryState>(GlossaryNotifier.new);
