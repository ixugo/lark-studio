class AppConfig {
  final PipelineConfig pipeline;
  final LLMConfig llm;
  final TTSConfig tts;
  final LipSyncConfig lipSync;

  const AppConfig({
    required this.pipeline,
    required this.llm,
    required this.tts,
    required this.lipSync,
  });

  factory AppConfig.fromJson(Map<String, dynamic> json) {
    return AppConfig(
      pipeline: PipelineConfig.fromJson(
        json['pipeline'] as Map<String, dynamic>? ?? {},
      ),
      llm: LLMConfig.fromJson(json['llm'] as Map<String, dynamic>? ?? {}),
      tts: TTSConfig.fromJson(json['tts'] as Map<String, dynamic>? ?? {}),
      lipSync: LipSyncConfig.fromJson(
        json['lip_sync'] as Map<String, dynamic>? ?? {},
      ),
    );
  }
}

class PipelineConfig {
  final int workers;
  final String whisperMode;
  final String whisperModel;
  final String ffmpegBin;
  final String defaultTargetLang;
  final String translatePrompt;
  final double maxSpeedFactor;
  final int translateChunkSize;
  final int ttsWorkers;
  final bool cleanIntermediate;
  final String subtitleOutput;

  const PipelineConfig({
    this.workers = 2,
    this.whisperMode = 'ffmpeg',
    this.whisperModel = '',
    this.ffmpegBin = '',
    this.defaultTargetLang = 'zh-CN',
    this.translatePrompt = '',
    this.maxSpeedFactor = 0,
    this.translateChunkSize = 10,
    this.ttsWorkers = 2,
    this.cleanIntermediate = false,
    this.subtitleOutput = 'burn',
  });

  factory PipelineConfig.fromJson(Map<String, dynamic> json) {
    return PipelineConfig(
      workers: _boundedInt(json['Workers'], fallback: 2, min: 1, max: 4),
      whisperMode: _knownString(
        json['WhisperMode'],
        fallback: 'whisper-cpp',
        values: const {'ffmpeg', 'whisper-cpp'},
      ),
      whisperModel: json['WhisperModel'] as String? ?? '',
      ffmpegBin: json['FFmpegBin'] as String? ?? '',
      defaultTargetLang: json['DefaultTargetLang'] as String? ?? 'zh-CN',
      translatePrompt: json['TranslatePrompt'] as String? ?? '',
      maxSpeedFactor: ((json['MaxSpeedFactor'] as num?)?.toDouble() ?? 0).clamp(
        0,
        1.5,
      ),
      translateChunkSize: _boundedInt(
        json['TranslateChunkSize'],
        fallback: 10,
        min: 5,
        max: 20,
      ),
      ttsWorkers: _boundedInt(json['TTSWorkers'], fallback: 2, min: 1, max: 4),
      cleanIntermediate: json['CleanIntermediate'] as bool? ?? false,
      subtitleOutput: _knownString(
        json['SubtitleOutput'],
        fallback: 'burn',
        values: const {'burn', 'file'},
      ),
    );
  }

  /// 旧配置可能保留零值，此处恢复为界面支持的默认范围。
  static int _boundedInt(
    dynamic value, {
    required int fallback,
    required int min,
    required int max,
  }) {
    final number = value is num ? value.toInt() : fallback;
    return number >= min && number <= max ? number : fallback;
  }

  /// 分段控件只接受已声明值，非法旧值统一回到默认项。
  static String _knownString(
    dynamic value, {
    required String fallback,
    required Set<String> values,
  }) {
    return value is String && values.contains(value) ? value : fallback;
  }
}

class LLMConfig {
  final String provider;
  final String baseUrl;
  final String apiKey;
  final String model;
  final String deepLXUrl;

  const LLMConfig({
    this.provider = 'bing',
    this.baseUrl = '',
    this.apiKey = '',
    this.model = '',
    this.deepLXUrl = '',
  });

  factory LLMConfig.fromJson(Map<String, dynamic> json) {
    return LLMConfig(
      provider: json['provider'] as String? ?? 'bing',
      baseUrl: json['base_url'] as String? ?? '',
      apiKey: json['api_key'] as String? ?? '',
      model: json['model'] as String? ?? '',
      deepLXUrl: json['deeplx_url'] as String? ?? '',
    );
  }
}

class TTSConfig {
  final String type;
  final String voice;
  final String baseUrl;
  final String apiKey;
  final String model;

  const TTSConfig({
    this.type = 'edge',
    this.voice = '',
    this.baseUrl = '',
    this.apiKey = '',
    this.model = '',
  });

  factory TTSConfig.fromJson(Map<String, dynamic> json) {
    return TTSConfig(
      type: json['Type'] as String? ?? 'edge',
      voice: json['Voice'] as String? ?? '',
      baseUrl: json['BaseURL'] as String? ?? '',
      apiKey: json['APIKey'] as String? ?? '',
      model: json['Model'] as String? ?? '',
    );
  }
}

class LipSyncConfig {
  final bool enabled;
  final String baseUrl;
  final String apiKey;

  const LipSyncConfig({
    this.enabled = false,
    this.baseUrl = '',
    this.apiKey = '',
  });

  factory LipSyncConfig.fromJson(Map<String, dynamic> json) {
    return LipSyncConfig(
      enabled: json['Enabled'] as bool? ?? false,
      baseUrl: json['BaseURL'] as String? ?? '',
      apiKey: json['APIKey'] as String? ?? '',
    );
  }
}
