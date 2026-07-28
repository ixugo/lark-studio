class Task {
  final String id;
  final String inputPath;
  final String outputDir;
  final int mode;
  final String sourceLang;
  final String targetLang;
  final String translator;
  final String outputContent;
  final String ttsEngine;
  final String ttsVoice;
  final double speechRate;
  final String subtitleOutput;
  final String recipeName;
  final int status;
  final String currentStep;
  final int progress;
  final int stepProgress;
  final String currentDetail;
  final DateTime? stepStartedAt;
  final String error;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Task({
    required this.id,
    required this.inputPath,
    required this.outputDir,
    required this.mode,
    required this.sourceLang,
    required this.targetLang,
    required this.translator,
    required this.outputContent,
    required this.ttsEngine,
    required this.ttsVoice,
    required this.speechRate,
    required this.subtitleOutput,
    required this.recipeName,
    required this.status,
    required this.currentStep,
    required this.progress,
    required this.stepProgress,
    required this.currentDetail,
    required this.stepStartedAt,
    required this.error,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Task.fromJson(Map<String, dynamic> json) {
    return Task(
      id: json['id'] as String? ?? '',
      inputPath: json['input_path'] as String? ?? '',
      outputDir: json['output_dir'] as String? ?? '',
      mode: json['mode'] as int? ?? 0,
      sourceLang: json['source_lang'] as String? ?? 'auto',
      targetLang: json['target_lang'] as String? ?? '',
      translator: json['translator'] as String? ?? 'bing',
      outputContent: json['output_content'] as String? ?? 'bilingual',
      ttsEngine: json['tts_engine'] as String? ?? 'edge',
      ttsVoice: json['tts_voice'] as String? ?? '',
      speechRate: (json['speech_rate'] as num?)?.toDouble() ?? 1,
      subtitleOutput: json['subtitle_output'] as String? ?? 'burn',
      recipeName: json['recipe_name'] as String? ?? '',
      status: json['status'] as int? ?? 0,
      currentStep: json['current_step'] as String? ?? '',
      progress: (json['progress'] as num?)?.toInt() ?? 0,
      stepProgress: (json['step_progress'] as num?)?.toInt() ?? 0,
      currentDetail: json['current_detail'] as String? ?? '',
      stepStartedAt: _parseOptionalTime(json['step_started_at']),
      error: json['error'] as String? ?? '',
      createdAt: _parseTime(json['created_at']),
      updatedAt: _parseTime(json['updated_at']),
    );
  }

  static DateTime _parseTime(dynamic v) {
    if (v is String && v.isNotEmpty) {
      return DateTime.tryParse(v) ?? DateTime.now();
    }
    return DateTime.now();
  }

  /// _parseOptionalTime 保留空时间，避免尚未开始的步骤显示虚假耗时。
  static DateTime? _parseOptionalTime(dynamic value) {
    if (value is String && value.isNotEmpty) {
      return DateTime.tryParse(value);
    }
    return null;
  }

  /// modeName 使用用户可读的完整模式名。
  String get modeName {
    switch (mode) {
      case 1:
        return '原文字幕';
      case 2:
        return '双语字幕';
      case 3:
        return '配音成片';
      default:
        return '未知';
    }
  }

  /// currentStepName 将流水线内部名称转换为纯中文文案。
  String get currentStepName => taskStepName(currentStep);

  /// relativeCreatedAt 返回任务创建至今的紧凑时间。
  String relativeCreatedAt([DateTime? now]) {
    return formatRelativeTime(createdAt, now: now);
  }

  // 0=待处理, 1=进行中, 2=已暂停, 3=已完成, 4=失败
  String get statusName {
    switch (status) {
      case 0:
        return '等待中';
      case 1:
        return '处理中';
      case 2:
        return '已暂停';
      case 3:
        return '已完成';
      case 4:
        return '失败';
      default:
        return '未知';
    }
  }

  bool get canPause => status == 1;
  bool get canResume => status == 2 || status == 4;

  String get fileName {
    final parts = inputPath.split('/');
    return parts.isNotEmpty ? parts.last : inputPath;
  }
}

/// TaskRecipe 保存可复用且会随任务写入 SQLite 的确定参数。
class TaskRecipe {
  final String name;
  final int mode;
  final String sourceLang;
  final String targetLang;
  final String translator;
  final String outputContent;
  final String ttsEngine;
  final String ttsVoice;
  final double speechRate;
  final String subtitleOutput;

  const TaskRecipe({
    this.name = '',
    this.mode = 3,
    this.sourceLang = 'auto',
    this.targetLang = 'zh-CN',
    this.translator = 'bing',
    this.outputContent = 'bilingual',
    this.ttsEngine = 'edge',
    this.ttsVoice = 'zh-CN-XiaoxiaoNeural',
    this.speechRate = 1,
    this.subtitleOutput = 'burn',
  });

  /// fromJson 从本地配方存储恢复参数。
  factory TaskRecipe.fromJson(Map<String, dynamic> json) {
    return TaskRecipe(
      name: json['name'] as String? ?? '',
      mode: (json['mode'] as num?)?.toInt() ?? 3,
      sourceLang: json['source_lang'] as String? ?? 'auto',
      targetLang: json['target_lang'] as String? ?? 'zh-CN',
      translator: json['translator'] as String? ?? 'bing',
      outputContent: json['output_content'] as String? ?? 'bilingual',
      ttsEngine: json['tts_engine'] as String? ?? 'edge',
      ttsVoice: json['tts_voice'] as String? ?? 'zh-CN-XiaoxiaoNeural',
      speechRate: (json['speech_rate'] as num?)?.toDouble() ?? 1,
      subtitleOutput: json['subtitle_output'] as String? ?? 'burn',
    );
  }

  /// toJson 同时用于本地配方持久化与任务创建请求。
  Map<String, dynamic> toJson() => {
    if (name.isNotEmpty) 'recipe_name': name,
    'mode': mode,
    'source_lang': sourceLang,
    'target_lang': targetLang,
    'translator': translator,
    'output_content': outputContent,
    'tts_engine': ttsEngine,
    'tts_voice': ttsVoice,
    'speech_rate': speechRate,
    'subtitle_output': subtitleOutput,
  };

  /// toStorageJson 使用稳定字段名保存配方名称。
  Map<String, dynamic> toStorageJson() => {...toJson(), 'name': name};
}

/// TaskStep 保存单个处理步骤的模型、进度与耗时。
class TaskStep {
  final String id;
  final String taskId;
  final String name;
  final int status;
  final int progress;
  final String detail;
  final String error;
  final DateTime? startedAt;
  final DateTime? endedAt;

  const TaskStep({
    required this.id,
    required this.taskId,
    required this.name,
    required this.status,
    required this.progress,
    required this.detail,
    required this.error,
    required this.startedAt,
    required this.endedAt,
  });

  /// fromJson 解析后端步骤记录。
  factory TaskStep.fromJson(Map<String, dynamic> json) {
    return TaskStep(
      id: json['id'] as String? ?? '',
      taskId: json['task_id'] as String? ?? '',
      name: json['name'] as String? ?? '',
      status: (json['status'] as num?)?.toInt() ?? 0,
      progress: (json['progress'] as num?)?.toInt() ?? 0,
      detail: json['detail'] as String? ?? '',
      error: json['error'] as String? ?? '',
      startedAt: _parseNullableTime(json['started_at']),
      endedAt: _parseNullableTime(json['ended_at']),
    );
  }

  /// elapsedText 以分秒显示已完成步骤的耗时。
  String get elapsedText {
    if (startedAt == null || endedAt == null) return '';
    final seconds = endedAt!.difference(startedAt!).inSeconds.clamp(0, 359999);
    final hours = seconds ~/ 3600;
    final minutes = (seconds % 3600) ~/ 60;
    final remain = seconds % 60;
    if (hours > 0) {
      return '$hours:${minutes.toString().padLeft(2, '0')}:${remain.toString().padLeft(2, '0')}';
    }
    return '${minutes.toString().padLeft(2, '0')}:${remain.toString().padLeft(2, '0')}';
  }
}

/// TaskLog 是按任务隔离的结构化实时日志。
class TaskLog {
  final int id;
  final String taskId;
  final String level;
  final String step;
  final String message;
  final DateTime createdAt;

  const TaskLog({
    required this.id,
    required this.taskId,
    required this.level,
    required this.step,
    required this.message,
    required this.createdAt,
  });

  /// fromJson 解析历史日志与 WebSocket 实时日志。
  factory TaskLog.fromJson(Map<String, dynamic> json) {
    return TaskLog(
      id: (json['id'] as num?)?.toInt() ?? 0,
      taskId: json['task_id'] as String? ?? '',
      level: json['level'] as String? ?? 'info',
      step: json['step'] as String? ?? '',
      message: json['message'] as String? ?? '',
      createdAt: _parseNullableTime(json['created_at']) ?? DateTime.now(),
    );
  }

  /// local 构造仅用于显示本地加载错误的日志。
  factory TaskLog.local(String taskId, String message) {
    return TaskLog(
      id: -1,
      taskId: taskId,
      level: 'error',
      step: '',
      message: message,
      createdAt: DateTime.now(),
    );
  }

  /// timeText 返回日志行的时分秒。
  String get timeText {
    final local = createdAt.toLocal();
    final hour = local.hour.toString().padLeft(2, '0');
    final minute = local.minute.toString().padLeft(2, '0');
    final second = local.second.toString().padLeft(2, '0');
    return '$hour:$minute:$second';
  }
}

/// taskStepName 将流水线步骤名统一转换为中文。
String taskStepName(String step) {
  return switch (step) {
    'whisper' => '听写',
    'split' => '语义分句',
    'translate' => '翻译',
    'tts' => '配音',
    'merge' => '音频合成',
    'lipsync' => '对口型',
    'burn' => '视频烧录',
    _ => '准备中',
  };
}

/// formatRelativeTime 生成适合任务列表的相对时间。
String formatRelativeTime(DateTime value, {DateTime? now}) {
  final current = now ?? DateTime.now();
  final elapsed = current.difference(value.toLocal());
  if (elapsed.isNegative || elapsed.inSeconds < 60) return '刚刚';
  if (elapsed.inMinutes < 60) return '${elapsed.inMinutes} 分钟前';
  if (elapsed.inHours < 24) return '${elapsed.inHours} 小时前';
  if (elapsed.inDays == 1) return '昨天';
  if (elapsed.inDays < 30) return '${elapsed.inDays} 天前';
  final local = value.toLocal();
  return '${local.year}-${local.month.toString().padLeft(2, '0')}-${local.day.toString().padLeft(2, '0')}';
}

/// _parseNullableTime 解析可为空的后端时间。
DateTime? _parseNullableTime(dynamic value) {
  if (value is String && value.isNotEmpty) return DateTime.tryParse(value);
  return null;
}
