class AppConfig {
  final PipelineConfig pipeline;
  final LLMConfig llm;
  final TTSConfig tts;

  const AppConfig({required this.pipeline, required this.llm, required this.tts});

  factory AppConfig.fromJson(Map<String, dynamic> json) {
    return AppConfig(
      pipeline: PipelineConfig.fromJson(json['pipeline'] as Map<String, dynamic>? ?? {}),
      llm: LLMConfig.fromJson(json['llm'] as Map<String, dynamic>? ?? {}),
      tts: TTSConfig.fromJson(json['tts'] as Map<String, dynamic>? ?? {}),
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
      workers: json['Workers'] as int? ?? 2,
      whisperMode: json['WhisperMode'] as String? ?? 'ffmpeg',
      whisperModel: json['WhisperModel'] as String? ?? '',
      ffmpegBin: json['FFmpegBin'] as String? ?? '',
      defaultTargetLang: json['DefaultTargetLang'] as String? ?? 'zh-CN',
      translatePrompt: json['TranslatePrompt'] as String? ?? '',
      maxSpeedFactor: (json['MaxSpeedFactor'] as num?)?.toDouble() ?? 0,
      translateChunkSize: json['TranslateChunkSize'] as int? ?? 10,
      ttsWorkers: json['TTSWorkers'] as int? ?? 2,
      cleanIntermediate: json['CleanIntermediate'] as bool? ?? false,
      subtitleOutput: json['SubtitleOutput'] as String? ?? 'burn',
    );
  }
}

class LLMConfig {
  final String baseUrl;
  final String apiKey;
  final String model;

  const LLMConfig({this.baseUrl = '', this.apiKey = '', this.model = ''});

  factory LLMConfig.fromJson(Map<String, dynamic> json) {
    return LLMConfig(
      baseUrl: json['base_url'] as String? ?? '',
      apiKey: json['api_key'] as String? ?? '',
      model: json['model'] as String? ?? '',
    );
  }
}

class TTSConfig {
  final String type;
  final String voice;
  final String baseUrl;
  final String apiKey;
  final String model;

  const TTSConfig({this.type = 'edge', this.voice = '', this.baseUrl = '', this.apiKey = '', this.model = ''});

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
