import 'package:flutter_test/flutter_test.dart';
import 'package:vdub_ui/data/models/task.dart';

/// main 验证任务面板依赖的进度、步骤、日志与时间文案。
void main() {
  group('任务模型', () {
    test('解析总进度与当前步骤详情', () {
      final task = Task.fromJson({
        'id': 'task-a',
        'input_path': '/tmp/demo.mp4',
        'mode': 3,
        'status': 1,
        'current_step': 'tts',
        'progress': 61,
        'step_progress': 24,
        'current_detail': 'edge-tts',
        'step_started_at': '2026-07-28T10:20:00Z',
        'created_at': '2026-07-28T10:15:00Z',
        'updated_at': '2026-07-28T10:20:00Z',
      });

      expect(task.modeName, '配音成片');
      expect(task.currentStepName, '配音');
      expect(task.progress, 61);
      expect(task.stepProgress, 24);
      expect(task.currentDetail, 'edge-tts');
      expect(task.stepStartedAt, isNotNull);
    });

    test('生成稳定的相对创建时间', () {
      final task = Task.fromJson({
        'created_at': '2026-07-28T10:15:00Z',
        'updated_at': '2026-07-28T10:15:00Z',
      });

      expect(
        task.relativeCreatedAt(DateTime.parse('2026-07-28T10:20:00Z')),
        '5 分钟前',
      );
    });

    test('配方可完整持久化并恢复', () {
      const recipe = TaskRecipe(
        name: '英文配音',
        sourceLang: 'en',
        targetLang: 'zh-CN',
        translator: 'bing',
        outputContent: 'translated',
        ttsEngine: 'edge',
        ttsVoice: 'zh-CN-XiaoxiaoNeural',
        speechRate: 1.2,
        subtitleOutput: 'none',
      );

      final restored = TaskRecipe.fromJson(recipe.toStorageJson());
      expect(restored.name, '英文配音');
      expect(restored.sourceLang, 'en');
      expect(restored.outputContent, 'translated');
      expect(restored.speechRate, 1.2);
      expect(restored.subtitleOutput, 'none');
    });
  });

  group('任务步骤与日志', () {
    test('解析步骤模型和完成耗时', () {
      final step = TaskStep.fromJson({
        'id': 'step-a',
        'task_id': 'task-a',
        'name': 'whisper',
        'status': 2,
        'progress': 100,
        'detail': 'large-v3',
        'started_at': '2026-07-28T10:20:00Z',
        'ended_at': '2026-07-28T10:23:42Z',
      });

      expect(step.detail, 'large-v3');
      expect(step.elapsedText, '03:42');
    });

    test('解析结构化日志时间与任务编号', () {
      final log = TaskLog.fromJson({
        'id': 7,
        'task_id': 'task-a',
        'level': 'success',
        'step': 'whisper',
        'message': '听写完成：387 条字幕',
        'created_at': '2026-07-28T10:23:47Z',
      });

      expect(log.id, 7);
      expect(log.taskId, 'task-a');
      expect(log.message, '听写完成：387 条字幕');
      expect(log.timeText, matches(RegExp(r'^\d{2}:\d{2}:\d{2}$')));
    });
  });
}
