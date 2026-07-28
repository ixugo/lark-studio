import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/task.dart';
import '../../../../data/repositories/task_repository.dart';
import '../../../../data/services/api_client.dart';
import '../../../../providers.dart';

/// 任务列表状态
class TaskListState {
  final List<Task> tasks;
  final bool loading;
  final String? error;

  const TaskListState({
    this.tasks = const [],
    this.loading = false,
    this.error,
  });

  TaskListState copyWith({
    List<Task>? tasks,
    bool? loading,
    String? error,
    bool clearError = false,
  }) {
    return TaskListState(
      tasks: tasks ?? this.tasks,
      loading: loading ?? this.loading,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

/// 任务列表状态管理
class TaskListNotifier extends Notifier<TaskListState> {
  late final TaskRepository _repo;

  @override
  TaskListState build() {
    _repo = ref.watch(taskRepoProvider);
    return const TaskListState();
  }

  Future<void> refresh() async {
    state = state.copyWith(loading: true, clearError: true);
    try {
      final tasks = await _repo.listTasks();
      state = state.copyWith(tasks: tasks, loading: false);
    } on ApiException catch (e) {
      state = state.copyWith(error: e.message, loading: false);
    } catch (_) {
      state = state.copyWith(error: '连接后端失败', loading: false);
    }
  }

  Future<void> createTask({
    required String inputPath,
    required int mode,
    String outputDir = '',
    String targetLang = '',
  }) async {
    await createRecipeTask(
      inputPath: inputPath,
      recipe: TaskRecipe(
        mode: mode,
        targetLang: targetLang.isEmpty ? 'zh-CN' : targetLang,
      ),
      outputDir: outputDir,
    );
  }

  /// createRecipeTask 使用任务链页面确定的完整配方创建任务。
  Future<void> createRecipeTask({
    required String inputPath,
    required TaskRecipe recipe,
    String outputDir = '',
  }) async {
    await _repo.createTask(
      inputPath: inputPath,
      recipe: recipe,
      outputDir: outputDir,
    );
    await refresh();
  }

  Future<void> deleteTask(String id) async {
    await _repo.deleteTask(id);
    await refresh();
  }

  Future<void> pauseTask(String id) async {
    await _repo.pauseTask(id);
    await refresh();
  }

  Future<void> resumeTask(String id) async {
    await _repo.resumeTask(id);
    await refresh();
  }

  Future<int> batchCreateTasks({
    required List<String> videos,
    required int mode,
    String targetLang = '',
  }) async {
    final recipe = TaskRecipe(
      mode: mode,
      targetLang: targetLang.isEmpty ? 'zh-CN' : targetLang,
    );
    final tasks = await _repo.batchCreateTasks(videos: videos, recipe: recipe);
    await refresh();
    return tasks.length;
  }
}

final taskListProvider = NotifierProvider<TaskListNotifier, TaskListState>(
  TaskListNotifier.new,
);
