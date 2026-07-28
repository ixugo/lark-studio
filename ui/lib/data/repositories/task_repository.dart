import '../models/task.dart';
import '../services/api_client.dart';

/// 任务数据仓库——封装 API 调用，提供领域模型给 ViewModel
class TaskRepository {
  final ApiClient _api;

  TaskRepository({required this._api});

  Future<List<Task>> listTasks({int page = 1, int size = 50}) =>
      _api.listTasks(page: page, size: size);

  Future<Task> getTask(String id) => _api.getTask(id);

  /// getTaskLogs 返回按时间正序排列的任务日志。
  Future<List<TaskLog>> getTaskLogs(String id) => _api.getTaskLogs(id);

  /// listTaskSteps 返回当前任务的步骤状态。
  Future<List<TaskStep>> listTaskSteps(String id) => _api.listTaskSteps(id);

  Future<Task> createTask({
    required String inputPath,
    required TaskRecipe recipe,
    String outputDir = '',
  }) => _api.createTask(
    inputPath: inputPath,
    recipe: recipe,
    outputDir: outputDir,
  );

  Future<void> deleteTask(String id) => _api.deleteTask(id);

  Future<void> pauseTask(String id) => _api.pauseTask(id);

  Future<Task> resumeTask(String id) => _api.resumeTask(id);

  Future<List<Task>> batchCreateTasks({
    required List<String> videos,
    required TaskRecipe recipe,
  }) => _api.batchCreateTasks(videos: videos, recipe: recipe);
}
