import '../models/task.dart';
import '../services/api_client.dart';

/// 任务数据仓库——封装 API 调用，提供领域模型给 ViewModel
class TaskRepository {
  final ApiClient _api;

  TaskRepository({required ApiClient api}) : _api = api;

  Future<List<Task>> listTasks({int page = 1, int size = 50}) =>
      _api.listTasks(page: page, size: size);

  Future<Task> getTask(String id) => _api.getTask(id);

  Future<Task> createTask({
    required String inputPath,
    required int mode,
    String outputDir = '',
    String targetLang = '',
  }) =>
      _api.createTask(
        inputPath: inputPath,
        mode: mode,
        outputDir: outputDir,
        targetLang: targetLang,
      );

  Future<void> deleteTask(String id) => _api.deleteTask(id);

  Future<void> pauseTask(String id) => _api.pauseTask(id);

  Future<Task> resumeTask(String id) => _api.resumeTask(id);

  Future<List<Task>> batchCreateTasks({
    required List<String> videos,
    required int mode,
    String targetLang = '',
  }) =>
      _api.batchCreateTasks(videos: videos, mode: mode, targetLang: targetLang);
}
