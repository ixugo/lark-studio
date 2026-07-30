import 'package:flutter_test/flutter_test.dart';
import 'package:vdub_ui/data/models/config.dart';

void main() {
  test('旧配置的零值会恢复为界面支持的默认值', () {
    final config = PipelineConfig.fromJson({
      'Workers': 0,
      'TranslateChunkSize': 0,
      'TTSWorkers': 0,
      'WhisperMode': '',
      'SubtitleOutput': '',
    });

    expect(config.workers, 2);
    expect(config.translateChunkSize, 10);
    expect(config.ttsWorkers, 2);
    expect(config.whisperMode, 'whisper-cpp');
    expect(config.subtitleOutput, 'burn');
  });

  test('旧版 FFmpeg 听写配置会迁移到 whisper.cpp', () {
    final config = PipelineConfig.fromJson({'WhisperMode': 'ffmpeg'});

    expect(config.whisperMode, 'whisper-cpp');
  });

  test('翻译配置默认使用必应并读取 DeepLX 地址', () {
    final defaults = LLMConfig.fromJson({});
    final configured = LLMConfig.fromJson({
      'provider': 'deeplx',
      'deeplx_url': 'http://127.0.0.1:1188/translate',
    });

    expect(defaults.provider, 'bing');
    expect(configured.provider, 'deeplx');
    expect(configured.deepLXUrl, 'http://127.0.0.1:1188/translate');
  });
}
