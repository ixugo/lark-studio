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
}
