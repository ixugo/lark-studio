import 'dart:ui';
import 'package:flutter/cupertino.dart'
    show
        CupertinoAlertDialog,
        CupertinoDialogAction,
        CupertinoSlider,
        CupertinoSlidingSegmentedControl,
        showCupertinoDialog;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../../data/models/config.dart';
import '../../../../providers.dart';
import '../../../core/app_colors.dart';
import '../view_models/settings_view_model.dart';

class SettingsPage extends HookConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final backend = ref.watch(backendProvider);
    useListenable(backend);
    final wsService = ref.watch(wsServiceProvider);
    useListenable(wsService);
    final ss = ref.watch(settingsProvider);
    final notifier = ref.read(settingsProvider.notifier);

    final llmBaseUrl = useTextEditingController();
    final llmApiKey = useTextEditingController();
    final llmModel = useTextEditingController();
    final whisperModel = useTextEditingController();
    final ffmpegBin = useTextEditingController();
    final targetLang = useTextEditingController();
    final translatePrompt = useTextEditingController();
    final lipSyncBaseUrl = useTextEditingController();
    final lipSyncApiKey = useTextEditingController();

    final workers = useState(2);
    final maxSpeedFactor = useState(0.0);
    final translateChunkSize = useState(10);
    final ttsWorkers = useState(2);
    final cleanIntermediate = useState(false);
    final subtitleOutput = useState('burn');
    final lipSyncEnabled = useState(false);

    void applyConfig(AppConfig cfg) {
      llmBaseUrl.text = cfg.llm.baseUrl;
      llmApiKey.text = cfg.llm.apiKey;
      llmModel.text = cfg.llm.model;
      whisperModel.text = cfg.pipeline.whisperModel;
      ffmpegBin.text = cfg.pipeline.ffmpegBin;
      targetLang.text = cfg.pipeline.defaultTargetLang;
      translatePrompt.text = cfg.pipeline.translatePrompt;
      workers.value = cfg.pipeline.workers.clamp(1, 4).toInt();
      maxSpeedFactor.value = cfg.pipeline.maxSpeedFactor
          .clamp(0, 1.5)
          .toDouble();
      translateChunkSize.value = cfg.pipeline.translateChunkSize
          .clamp(5, 20)
          .toInt();
      ttsWorkers.value = cfg.pipeline.ttsWorkers.clamp(1, 4).toInt();
      cleanIntermediate.value = cfg.pipeline.cleanIntermediate;
      subtitleOutput.value = cfg.pipeline.subtitleOutput;
      lipSyncEnabled.value = cfg.lipSync.enabled;
      lipSyncBaseUrl.text = cfg.lipSync.baseUrl;
      lipSyncApiKey.text = cfg.lipSync.apiKey;
    }

    useEffect(() {
      Future.microtask(() async {
        await notifier.loadConfig();
        final cfg = ref.read(settingsProvider).config;
        if (cfg != null) applyConfig(cfg);
      });
      return null;
    }, const []);

    ref.listen<SettingsState>(settingsProvider, (prev, next) {
      if (next.config != null && prev?.config != next.config) {
        applyConfig(next.config!);
      }
    });

    Future<void> saveConfig() async {
      try {
        final updates = <String, dynamic>{
          'llm': {
            'base_url': llmBaseUrl.text.trim(),
            if (!llmApiKey.text.contains('****'))
              'api_key': llmApiKey.text.trim(),
            'model': llmModel.text.trim(),
          },
          'pipeline': {
            'workers': workers.value,
            'whisper_model': whisperModel.text.trim(),
            'ffmpeg_bin': ffmpegBin.text.trim(),
            'default_target_lang': targetLang.text.trim(),
            'translate_prompt': translatePrompt.text.trim(),
            'max_speed_factor': maxSpeedFactor.value,
            'translate_chunk_size': translateChunkSize.value,
            'tts_workers': ttsWorkers.value,
            'clean_intermediate': cleanIntermediate.value,
            'subtitle_output': subtitleOutput.value,
          },
          'lip_sync': {
            'enabled': lipSyncEnabled.value,
            'base_url': lipSyncBaseUrl.text.trim(),
            if (!lipSyncApiKey.text.contains('****'))
              'api_key': lipSyncApiKey.text.trim(),
          },
        };
        final cfg = await notifier.saveConfig(updates);
        if (cfg != null) applyConfig(cfg);
        if (context.mounted) _showToast(context, '配置已保存');
      } catch (e) {
        if (context.mounted) _showToast(context, '保存失败: $e');
      }
    }

    return Column(
      children: [
        _SettingsToolbar(
          saving: ss.saving,
          canSave: ss.config != null,
          onSave: saveConfig,
        ),
        Expanded(
          child: ss.loading
              ? const Center(child: ProgressCircle(radius: 14))
              : ss.error != null && ss.config == null
              ? Center(
                  child: Text(
                    ss.error!,
                    style: TextStyle(
                      color: AppColors.of(context).textSecondary,
                    ),
                  ),
                )
              : _buildForm(
                  context,
                  backend,
                  wsService,
                  ss,
                  workers,
                  maxSpeedFactor,
                  translateChunkSize,
                  ttsWorkers,
                  cleanIntermediate,
                  subtitleOutput,
                  lipSyncEnabled,
                  llmBaseUrl,
                  llmApiKey,
                  llmModel,
                  whisperModel,
                  ffmpegBin,
                  targetLang,
                  translatePrompt,
                  lipSyncBaseUrl,
                  lipSyncApiKey,
                  notifier,
                ),
        ),
      ],
    );
  }

  Widget _buildForm(
    BuildContext context,
    dynamic backend,
    dynamic wsService,
    SettingsState ss,
    ValueNotifier<int> workers,
    ValueNotifier<double> maxSpeedFactor,
    ValueNotifier<int> translateChunkSize,
    ValueNotifier<int> ttsWorkers,
    ValueNotifier<bool> cleanIntermediate,
    ValueNotifier<String> subtitleOutput,
    ValueNotifier<bool> lipSyncEnabled,
    TextEditingController llmBaseUrl,
    TextEditingController llmApiKey,
    TextEditingController llmModel,
    TextEditingController whisperModel,
    TextEditingController ffmpegBin,
    TextEditingController targetLang,
    TextEditingController translatePrompt,
    TextEditingController lipSyncBaseUrl,
    TextEditingController lipSyncApiKey,
    SettingsNotifier notifier,
  ) {
    final c = AppColors.of(context);
    return ListView(
      padding: const EdgeInsets.fromLTRB(28, 20, 28, 40),
      children: [
        _GlassSection(
          title: '连接状态',
          children: [
            _StatusRow(
              label: '引擎',
              online: backend.online,
              detail: backend.online ? '端口 ${backend.port}' : null,
            ),
            _StatusRow(label: 'WebSocket', online: wsService.connected),
          ],
        ),
        const SizedBox(height: 20),
        _GlassSection(
          title: 'LLM 翻译',
          children: [
            _GlassField(
              label: 'API 地址',
              controller: llmBaseUrl,
              placeholder: 'http://localhost:11434/v1',
            ),
            _GlassField(
              label: 'API 密钥',
              controller: llmApiKey,
              placeholder: 'sk-xxx',
              obscure: true,
            ),
            _GlassField(
              label: '模型名称',
              controller: llmModel,
              placeholder: 'qwen2.5:7b',
            ),
          ],
        ),
        const SizedBox(height: 20),
        _GlassSection(
          title: '对口型 (MuseTalk)',
          children: [
            _switchRow(
              '启用',
              lipSyncEnabled.value,
              '配音后自动对口型（需 MuseTalk 服务）',
              (v) => lipSyncEnabled.value = v,
              c,
            ),
            if (lipSyncEnabled.value) ...[
              _GlassField(
                label: 'API 地址',
                controller: lipSyncBaseUrl,
                placeholder: 'http://localhost:7860',
              ),
              _GlassField(
                label: 'API 密钥',
                controller: lipSyncApiKey,
                placeholder: '选填',
                obscure: true,
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(18, 0, 18, 12),
                child: Text(
                  '需自行部署 MuseTalk 服务。配音完成后自动调用 API 做唇形同步。'
                  '也可连接阿里云万象大模型等兼容服务。',
                  style: TextStyle(fontSize: 11, color: c.textSecondary),
                ),
              ),
            ],
          ],
        ),
        const SizedBox(height: 20),
        _GlassSection(
          title: '流水线',
          children: [
            const Padding(
              padding: EdgeInsets.fromLTRB(18, 14, 18, 2),
              child: Row(
                children: [Text('听写引擎'), Spacer(), Text('whisper.cpp')],
              ),
            ),
            _GlassField(
              label: '模型路径',
              controller: whisperModel,
              placeholder: '/path/to/ggml-large-v3.bin',
            ),
            if (ss.models.isNotEmpty)
              Padding(
                padding: const EdgeInsets.fromLTRB(18, 0, 18, 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '可用模型（下载完成后点“使用”即生效）',
                      style: TextStyle(fontSize: 11, color: c.textSecondary),
                    ),
                    const SizedBox(height: 6),
                    ...ss.models.map(
                      (m) => _ModelRow(
                        name: m['name'] as String? ?? '',
                        size: m['size'] as String? ?? '',
                        desc: m['desc'] as String? ?? '',
                        downloaded: m['downloaded'] as bool? ?? false,
                        downloading: m['downloading'] as bool? ?? false,
                        progress: m['progress'] as int? ?? 0,
                        path: m['path'] as String? ?? '',
                        onDownload: () =>
                            notifier.downloadModel(m['name'] as String? ?? ''),
                        onSelect: () async {
                          final p = m['path'] as String? ?? '';
                          if (p.isEmpty) {
                            return;
                          }
                          whisperModel.text = p;
                          final cfg = await notifier.saveConfig({
                            'pipeline': {'whisper_model': p},
                          });
                          if (cfg != null) {
                            whisperModel.text = cfg.pipeline.whisperModel;
                          }
                        },
                      ),
                    ),
                  ],
                ),
              ),
            _GlassField(
              label: 'FFmpeg',
              controller: ffmpegBin,
              placeholder: '留空使用 PATH',
            ),
            _GlassField(
              label: '目标语言',
              controller: targetLang,
              placeholder: 'zh-CN',
            ),
            _sliderRow(
              'Worker 数',
              workers.value,
              1,
              4,
              (v) => workers.value = v,
              c,
            ),
            _doubleSliderRow(
              '调速上限',
              maxSpeedFactor.value,
              0,
              1.5,
              '不调速',
              (v) => maxSpeedFactor.value = double.parse(v.toStringAsFixed(1)),
              c,
            ),
            _sliderRow(
              '翻译分块',
              translateChunkSize.value,
              5,
              20,
              (v) => translateChunkSize.value = v,
              c,
            ),
            _sliderRow(
              'TTS 并发',
              ttsWorkers.value,
              1,
              4,
              (v) => ttsWorkers.value = v,
              c,
            ),
            _segmentRow(
              '字幕输出',
              {'burn': '烧录到视频', 'file': '独立字幕文件'},
              subtitleOutput.value,
              (v) => subtitleOutput.value = v,
              c,
            ),
            _switchRow(
              '清理中间产物',
              cleanIntermediate.value,
              '完成后删除 raw.mp3/audio_segs 等临时文件',
              (v) => cleanIntermediate.value = v,
              c,
            ),
          ],
        ),
        const SizedBox(height: 20),
        _GlassSection(
          title: '翻译提示词',
          children: [
            _GlassMultiLine(
              controller: translatePrompt,
              placeholder: '留空使用内置默认模板。\n可用变量: {{target_lang}} {{count}}',
            ),
          ],
        ),
      ],
    );
  }

  Widget _segmentRow(
    String label,
    Map<String, String> options,
    String value,
    ValueChanged<String> onChanged,
    AppColors c,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: TextStyle(fontSize: 13, color: c.textPrimary),
            ),
          ),
          Expanded(
            child: CupertinoSlidingSegmentedControl<String>(
              groupValue: value,
              children: options.map(
                (k, v) => MapEntry(
                  k,
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                    child: Text(v, style: const TextStyle(fontSize: 12)),
                  ),
                ),
              ),
              onValueChanged: (v) {
                if (v != null) onChanged(v);
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _doubleSliderRow(
    String label,
    double value,
    double min,
    double max,
    String hint,
    ValueChanged<double> onChanged,
    AppColors c,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: TextStyle(fontSize: 13, color: c.textPrimary),
            ),
          ),
          Expanded(
            child: CupertinoSlider(
              value: value,
              min: min,
              max: max,
              divisions: ((max - min) * 10).round(),
              onChanged: onChanged,
            ),
          ),
          SizedBox(
            width: 56,
            child: Text(
              value <= 1 ? hint : '${value.toStringAsFixed(1)}x',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: c.textPrimary,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _sliderRow(
    String label,
    int value,
    int min,
    int max,
    ValueChanged<int> onChanged,
    AppColors c,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: TextStyle(fontSize: 13, color: c.textPrimary),
            ),
          ),
          Expanded(
            child: CupertinoSlider(
              value: value.toDouble(),
              min: min.toDouble(),
              max: max.toDouble(),
              divisions: max - min,
              onChanged: (v) => onChanged(v.round()),
            ),
          ),
          SizedBox(
            width: 30,
            child: Text(
              '$value',
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w500,
                color: c.textPrimary,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _switchRow(
    String label,
    bool value,
    String hint,
    ValueChanged<bool> onChanged,
    AppColors c,
  ) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  label,
                  style: TextStyle(fontSize: 13, color: c.textPrimary),
                ),
                const SizedBox(height: 2),
                Text(
                  hint,
                  style: TextStyle(fontSize: 11, color: c.textSecondary),
                ),
              ],
            ),
          ),
          MacosSwitch(value: value, onChanged: onChanged),
        ],
      ),
    );
  }

  void _showToast(BuildContext context, String msg) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        content: Text(msg),
        actions: [
          CupertinoDialogAction(
            child: const Text('确定'),
            onPressed: () => Navigator.pop(context),
          ),
        ],
      ),
    );
  }
}

class _SettingsToolbar extends StatelessWidget {
  final bool saving;
  final bool canSave;
  final VoidCallback onSave;
  const _SettingsToolbar({
    required this.saving,
    required this.canSave,
    required this.onSave,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          height: 52,
          padding: const EdgeInsets.symmetric(horizontal: 28),
          decoration: BoxDecoration(
            color: c.barBg.withValues(alpha: 0.9),
            border: Border(
              bottom: BorderSide(color: c.borderLight, width: 0.5),
            ),
          ),
          child: Row(
            children: [
              Text(
                '设置',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: c.textPrimary,
                  letterSpacing: -0.5,
                ),
              ),
              const Spacer(),
              if (canSave)
                PushButton(
                  controlSize: ControlSize.regular,
                  color: AppColors.blue,
                  onPressed: saving ? null : onSave,
                  child: Text(
                    saving ? '保存中...' : '保存配置',
                    style: const TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: Color(0xFFFFFFFF),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _GlassSection extends StatelessWidget {
  final String title;
  final List<Widget> children;
  const _GlassSection({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 6, bottom: 8),
          child: Text(
            title,
            style: TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: c.textSecondary,
              letterSpacing: 0.3,
            ),
          ),
        ),
        ClipRRect(
          borderRadius: BorderRadius.circular(14),
          child: BackdropFilter(
            filter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
            child: Container(
              decoration: BoxDecoration(
                color: c.cardBg.withValues(alpha: 0.85),
                borderRadius: BorderRadius.circular(14),
                border: Border.all(
                  color: c.borderLight.withValues(alpha: 0.3),
                  width: 0.5,
                ),
              ),
              child: Column(children: children),
            ),
          ),
        ),
      ],
    );
  }
}

class _GlassField extends StatelessWidget {
  final String label;
  final TextEditingController controller;
  final String placeholder;
  final bool obscure;
  const _GlassField({
    required this.label,
    required this.controller,
    this.placeholder = '',
    this.obscure = false,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(
            width: 100,
            child: Text(
              label,
              style: TextStyle(fontSize: 13, color: c.textPrimary),
            ),
          ),
          Expanded(
            child: MacosTextField(
              controller: controller,
              placeholder: placeholder,
              obscureText: obscure,
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              style: TextStyle(fontSize: 13, color: c.textPrimary),
              decoration: BoxDecoration(
                color: c.inputBg,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: c.borderLight),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _GlassMultiLine extends StatelessWidget {
  final TextEditingController controller;
  final String placeholder;
  const _GlassMultiLine({required this.controller, this.placeholder = ''});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
      child: MacosTextField(
        controller: controller,
        placeholder: placeholder,
        maxLines: 6,
        minLines: 3,
        padding: const EdgeInsets.all(12),
        style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
        decoration: BoxDecoration(
          color: const Color(0xFF000000).withValues(alpha: 0.03),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
            color: const Color(0xFF000000).withValues(alpha: 0.06),
          ),
        ),
      ),
    );
  }
}

class _ModelRow extends StatelessWidget {
  final String name, size, desc, path;
  final bool downloaded, downloading;
  final int progress;
  final VoidCallback onDownload, onSelect;
  const _ModelRow({
    required this.name,
    required this.size,
    required this.desc,
    required this.downloaded,
    required this.downloading,
    required this.progress,
    required this.path,
    required this.onDownload,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  name,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: c.textPrimary,
                  ),
                ),
                Text(
                  '$size · $desc',
                  style: TextStyle(fontSize: 10, color: c.textSecondary),
                ),
              ],
            ),
          ),
          if (downloaded)
            PushButton(
              controlSize: ControlSize.mini,
              secondary: true,
              onPressed: onSelect,
              child: const Text(
                '使用',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: AppColors.green,
                ),
              ),
            )
          else if (downloading)
            Text(
              '$progress%',
              style: const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: AppColors.blue,
              ),
            )
          else
            PushButton(
              controlSize: ControlSize.mini,
              secondary: true,
              onPressed: onDownload,
              child: const Text(
                '下载',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: AppColors.blue,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _StatusRow extends StatelessWidget {
  final String label;
  final bool online;
  final String? detail;
  const _StatusRow({required this.label, required this.online, this.detail});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 11),
      child: Row(
        children: [
          Text(label, style: TextStyle(fontSize: 13, color: c.textPrimary)),
          const Spacer(),
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: online ? AppColors.green : AppColors.red,
              boxShadow: [
                BoxShadow(
                  color: (online ? AppColors.green : AppColors.red).withValues(
                    alpha: 0.35,
                  ),
                  blurRadius: 6,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            online ? (detail ?? '已连接') : '未连接',
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: online ? AppColors.green : AppColors.red,
            ),
          ),
        ],
      ),
    );
  }
}
