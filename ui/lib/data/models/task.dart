class Task {
  final String id;
  final String inputPath;
  final String outputDir;
  final int mode;
  final String targetLang;
  final int status;
  final String currentStep;
  final int progress;
  final String error;
  final DateTime createdAt;
  final DateTime updatedAt;

  const Task({
    required this.id,
    required this.inputPath,
    required this.outputDir,
    required this.mode,
    required this.targetLang,
    required this.status,
    required this.currentStep,
    required this.progress,
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
      targetLang: json['target_lang'] as String? ?? '',
      status: json['status'] as int? ?? 0,
      currentStep: json['current_step'] as String? ?? '',
      progress: json['progress'] as int? ?? 0,
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

  String get modeName {
    switch (mode) {
      case 1:
        return '字幕';
      case 2:
        return '翻译';
      case 3:
        return '配音';
      default:
        return '未知';
    }
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
