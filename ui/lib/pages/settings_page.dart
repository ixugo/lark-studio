import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../models/config.dart';
import '../services/api_client.dart';
import '../services/backend_service.dart';
import '../services/websocket_service.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  AppConfig? _config;
  bool _loading = true;
  bool _saving = false;
  String? _error;

  List<Map<String, dynamic>> _terms = [];
  final _newTermController = TextEditingController();
  final _newTransController = TextEditingController();

  final _llmBaseUrl = TextEditingController();
  final _llmApiKey = TextEditingController();
  final _llmModel = TextEditingController();

  String _ttsType = 'edge';
  final _ttsVoice = TextEditingController();
  final _ttsBaseUrl = TextEditingController();
  final _ttsApiKey = TextEditingController();
  final _ttsModel = TextEditingController();

  final _whisperModel = TextEditingController();
  final _ffmpegBin = TextEditingController();
  final _targetLang = TextEditingController();
  final _translatePrompt = TextEditingController();
  String _whisperMode = 'ffmpeg';
  int _workers = 2;
  double _maxSpeedFactor = 0;
  int _translateChunkSize = 10;
  int _ttsWorkers = 2;
  bool _cleanIntermediate = false;

  @override
  void initState() {
    super.initState();
    _loadConfig();
  }

  @override
  void dispose() {
    _newTermController.dispose();
    _newTransController.dispose();
    _llmBaseUrl.dispose();
    _llmApiKey.dispose();
    _llmModel.dispose();
    _ttsVoice.dispose();
    _ttsBaseUrl.dispose();
    _ttsApiKey.dispose();
    _ttsModel.dispose();
    _whisperModel.dispose();
    _ffmpegBin.dispose();
    _targetLang.dispose();
    _translatePrompt.dispose();
    super.dispose();
  }

  Future<void> _loadConfig() async {
    setState(() { _loading = true; _error = null; });
    try {
      final api = context.read<ApiClient>();
      final cfg = await api.getConfig();
      _applyConfig(cfg);
      try { _terms = await api.listTerms(); } catch (_) {}
      setState(() { _config = cfg; _loading = false; });
    } catch (e) {
      setState(() { _error = '$e'; _loading = false; });
    }
  }

  Future<void> _addTerm() async {
    final text = _newTermController.text.trim();
    if (text.isEmpty) return;
    final trans = _newTransController.text.trim();
    final api = context.read<ApiClient>();
    try {
      await api.createTerm(text, translation: trans);
      _newTermController.clear();
      _newTransController.clear();
      _terms = await api.listTerms();
      if (mounted) setState(() {});
    } catch (e) {
      if (mounted) _showToast('添加失败: $e');
    }
  }

  Future<void> _deleteTerm(int id) async {
    final api = context.read<ApiClient>();
    try {
      await api.deleteTerm(id);
      _terms = await api.listTerms();
      if (mounted) setState(() {});
    } catch (e) {
      if (mounted) _showToast('删除失败: $e');
    }
  }

  void _applyConfig(AppConfig cfg) {
    _llmBaseUrl.text = cfg.llm.baseUrl;
    _llmApiKey.text = cfg.llm.apiKey;
    _llmModel.text = cfg.llm.model;
    _ttsType = cfg.tts.type;
    _ttsVoice.text = cfg.tts.voice;
    _ttsBaseUrl.text = cfg.tts.baseUrl;
    _ttsApiKey.text = cfg.tts.apiKey;
    _ttsModel.text = cfg.tts.model;
    _whisperMode = cfg.pipeline.whisperMode;
    _whisperModel.text = cfg.pipeline.whisperModel;
    _ffmpegBin.text = cfg.pipeline.ffmpegBin;
    _targetLang.text = cfg.pipeline.defaultTargetLang;
    _translatePrompt.text = cfg.pipeline.translatePrompt;
    _workers = cfg.pipeline.workers;
    _maxSpeedFactor = cfg.pipeline.maxSpeedFactor;
    _translateChunkSize = cfg.pipeline.translateChunkSize;
    _ttsWorkers = cfg.pipeline.ttsWorkers;
    _cleanIntermediate = cfg.pipeline.cleanIntermediate;
  }

  Future<void> _saveConfig() async {
    setState(() => _saving = true);
    try {
      final updates = <String, dynamic>{
        'llm': {
          'base_url': _llmBaseUrl.text.trim(),
          if (!_llmApiKey.text.contains('****')) 'api_key': _llmApiKey.text.trim(),
          'model': _llmModel.text.trim(),
        },
        'tts': {
          'type': _ttsType,
          'voice': _ttsVoice.text.trim(),
          'base_url': _ttsBaseUrl.text.trim(),
          if (!_ttsApiKey.text.contains('****')) 'api_key': _ttsApiKey.text.trim(),
          'model': _ttsModel.text.trim(),
        },
        'pipeline': {
          'workers': _workers,
          'whisper_mode': _whisperMode,
          'whisper_model': _whisperModel.text.trim(),
          'ffmpeg_bin': _ffmpegBin.text.trim(),
          'default_target_lang': _targetLang.text.trim(),
          'translate_prompt': _translatePrompt.text.trim(),
          'max_speed_factor': _maxSpeedFactor,
          'translate_chunk_size': _translateChunkSize,
          'tts_workers': _ttsWorkers,
          'clean_intermediate': _cleanIntermediate,
        },
      };
      final cfg = await context.read<ApiClient>().updateConfig(updates);
      _applyConfig(cfg);
      setState(() { _config = cfg; _saving = false; });
      if (mounted) _showToast('配置已保存');
    } catch (e) {
      setState(() => _saving = false);
      if (mounted) _showToast('保存失败: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    final backend = context.watch<BackendService>();
    final wsService = context.watch<WebSocketService>();

    return Column(
      children: [
        _SettingsToolbar(saving: _saving, canSave: _config != null, onSave: _saveConfig),
        Expanded(
          child: _loading
              ? const Center(child: CupertinoActivityIndicator(radius: 14))
              : _error != null && _config == null
                  ? Center(child: Text(_error!, style: const TextStyle(color: Color(0xFF8E8E93))))
                  : _buildForm(backend, wsService),
        ),
      ],
    );
  }

  Widget _buildForm(BackendService backend, WebSocketService wsService) {
    return ListView(
      padding: const EdgeInsets.fromLTRB(28, 20, 28, 40),
      children: [
        _GlassSection(title: '连接状态', children: [
          _StatusRow(label: '引擎', online: backend.online, detail: backend.online ? '端口 ${backend.port}' : null),
          _StatusRow(label: 'WebSocket', online: wsService.connected),
        ]),
        const SizedBox(height: 20),
        _GlassSection(title: 'LLM 翻译', children: [
          _GlassField(label: 'API 地址', controller: _llmBaseUrl, placeholder: 'http://localhost:11434/v1'),
          _GlassField(label: 'API 密钥', controller: _llmApiKey, placeholder: 'sk-xxx', obscure: true),
          _GlassField(label: '模型名称', controller: _llmModel, placeholder: 'qwen2.5:7b'),
        ]),
        const SizedBox(height: 20),
        _GlassSection(title: 'TTS 语音合成', children: [
          _segmentRow('TTS 类型', {'edge': 'Edge TTS', 'openai': 'OpenAI TTS'}, _ttsType, (v) => setState(() => _ttsType = v)),
          _GlassField(label: '语音名称', controller: _ttsVoice, placeholder: 'zh-CN-YunjianNeural'),
          if (_ttsType == 'openai') ...[
            _GlassField(label: 'API 地址', controller: _ttsBaseUrl, placeholder: 'https://api.openai.com/v1'),
            _GlassField(label: 'API 密钥', controller: _ttsApiKey, placeholder: 'sk-xxx', obscure: true),
            _GlassField(label: '模型', controller: _ttsModel, placeholder: 'tts-1'),
            const Padding(
              padding: EdgeInsets.fromLTRB(18, 0, 18, 12),
              child: Text(
                '支持 CosyVoice / F5-TTS / ChatTTS 等开源 TTS，'
                '部署后将 API 地址指向本地服务即可 (如 http://localhost:8880/v1)。',
                style: TextStyle(fontSize: 11, color: Color(0xFF8E8E93)),
              ),
            ),
          ],
        ]),
        const SizedBox(height: 20),
        _GlassSection(title: '流水线', children: [
          _segmentRow('Whisper', {'ffmpeg': 'FFmpeg', 'whisper-cpp': 'whisper.cpp'}, _whisperMode, (v) => setState(() => _whisperMode = v)),
          _GlassField(label: '模型路径', controller: _whisperModel, placeholder: '/path/to/ggml-large-v3.bin'),
          _GlassField(label: 'FFmpeg', controller: _ffmpegBin, placeholder: '留空使用 PATH'),
          _GlassField(label: '目标语言', controller: _targetLang, placeholder: 'zh-CN'),
          _sliderRow('Worker 数', _workers, 1, 4, (v) => setState(() => _workers = v)),
          _doubleSliderRow('调速上限', _maxSpeedFactor, 0, 1.5, '不调速',
            (v) => setState(() => _maxSpeedFactor = double.parse(v.toStringAsFixed(1)))),
          _sliderRow('翻译分块', _translateChunkSize, 5, 20,
            (v) => setState(() => _translateChunkSize = v)),
          _sliderRow('TTS 并发', _ttsWorkers, 1, 4,
            (v) => setState(() => _ttsWorkers = v)),
          _switchRow('清理中间产物', _cleanIntermediate, '完成后删除 raw.mp3/audio_segs 等临时文件',
            (v) => setState(() => _cleanIntermediate = v)),
        ]),
        const SizedBox(height: 20),
        _GlassSection(title: '翻译提示词', children: [
          _GlassMultiLine(
            controller: _translatePrompt,
            placeholder: '留空使用内置默认模板。\n可用变量: {{target_lang}} {{count}}',
          ),
        ]),
        const SizedBox(height: 20),
        _GlassSection(title: '术语锁定', children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 12, 18, 6),
            child: Text('翻译时保持原文不翻译的专有名词（不区分大小写）',
              style: TextStyle(fontSize: 11, color: const Color(0xFF8E8E93))),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 6),
            child: Row(
              children: [
                Expanded(
                  flex: 3,
                  child: CupertinoTextField(
                    controller: _newTermController,
                    placeholder: '源词 (如 Golang)',
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    style: const TextStyle(fontSize: 13),
                    onSubmitted: (_) => _addTerm(),
                    decoration: BoxDecoration(
                      color: const Color(0xFF000000).withValues(alpha: 0.03),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0xFF000000).withValues(alpha: 0.06)),
                    ),
                  ),
                ),
                const Padding(
                  padding: EdgeInsets.symmetric(horizontal: 6),
                  child: Text('→', style: TextStyle(fontSize: 14, color: Color(0xFF8E8E93))),
                ),
                Expanded(
                  flex: 3,
                  child: CupertinoTextField(
                    controller: _newTransController,
                    placeholder: '译文 (留空=保持原文)',
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    style: const TextStyle(fontSize: 13),
                    onSubmitted: (_) => _addTerm(),
                    decoration: BoxDecoration(
                      color: const Color(0xFF000000).withValues(alpha: 0.03),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0xFF000000).withValues(alpha: 0.06)),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                CupertinoButton(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                  color: const Color(0xFF007AFF),
                  borderRadius: BorderRadius.circular(8),
                  minimumSize: Size.zero,
                  onPressed: _addTerm,
                  child: const Text('添加', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: CupertinoColors.white)),
                ),
              ],
            ),
          ),
          if (_terms.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(18, 4, 18, 12),
              child: Wrap(
                spacing: 6,
                runSpacing: 6,
                children: _terms.map((t) {
                  final id = t['id'] as int;
                  final text = t['text'] as String? ?? '';
                  final trans = t['translation'] as String? ?? text;
                  final label = text == trans ? text : '$text → $trans';
                  return _TermChip(text: label, onDelete: () => _deleteTerm(id));
                }).toList(),
              ),
            ),
          if (_terms.isEmpty)
            const Padding(
              padding: EdgeInsets.only(bottom: 12),
              child: Center(child: Text('暂无术语', style: TextStyle(fontSize: 12, color: Color(0xFFAEAEB2)))),
            ),
        ]),
      ],
    );
  }

  Widget _segmentRow(String label, Map<String, String> options, String value, ValueChanged<String> onChanged) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 100, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
          Expanded(
            child: CupertinoSlidingSegmentedControl<String>(
              groupValue: value,
              children: options.map((k, v) => MapEntry(k, Padding(
                padding: const EdgeInsets.symmetric(horizontal: 8),
                child: Text(v, style: const TextStyle(fontSize: 12)),
              ))),
              onValueChanged: (v) { if (v != null) onChanged(v); },
            ),
          ),
        ],
      ),
    );
  }

  Widget _doubleSliderRow(String label, double value, double min, double max, String hint, ValueChanged<double> onChanged) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 100, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
          Expanded(
            child: CupertinoSlider(
              value: value, min: min, max: max,
              divisions: ((max - min) * 10).round(),
              onChanged: onChanged,
            ),
          ),
          SizedBox(width: 56, child: Text(value <= 1 ? hint : '${value.toStringAsFixed(1)}x',
            style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF3A3A3C)))),
        ],
      ),
    );
  }

  Widget _sliderRow(String label, int value, int min, int max, ValueChanged<int> onChanged) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 100, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
          Expanded(
            child: CupertinoSlider(
              value: value.toDouble(), min: min.toDouble(), max: max.toDouble(),
              divisions: max - min,
              onChanged: (v) => onChanged(v.round()),
            ),
          ),
          SizedBox(width: 30, child: Text('$value', style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF3A3A3C)))),
        ],
      ),
    );
  }

  Widget _switchRow(String label, bool value, String hint, ValueChanged<bool> onChanged) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C))),
                const SizedBox(height: 2),
                Text(hint, style: const TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
              ],
            ),
          ),
          CupertinoSwitch(value: value, onChanged: onChanged),
        ],
      ),
    );
  }

  void _showToast(String msg) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        content: Text(msg),
        actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))],
      ),
    );
  }
}

class _SettingsToolbar extends StatelessWidget {
  final bool saving;
  final bool canSave;
  final VoidCallback onSave;

  const _SettingsToolbar({required this.saving, required this.canSave, required this.onSave});

  @override
  Widget build(BuildContext context) {
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          height: 52,
          padding: const EdgeInsets.symmetric(horizontal: 28),
          decoration: BoxDecoration(
            color: const Color(0xFFFFFFFF).withValues(alpha: 0.65),
            border: const Border(bottom: BorderSide(color: Color(0x1A000000), width: 0.5)),
          ),
          child: Row(
            children: [
              const Text('设置', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700, color: Color(0xFF1D1D1F), letterSpacing: -0.5)),
              const Spacer(),
              if (canSave)
                CupertinoButton.filled(
                  padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 6),
                  borderRadius: BorderRadius.circular(10),
                  onPressed: saving ? null : onSave,
                  child: Text(saving ? '保存中...' : '保存配置',
                    style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: CupertinoColors.white)),
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
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 6, bottom: 8),
          child: Text(title, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: Color(0xFF8E8E93), letterSpacing: 0.3)),
        ),
        ClipRRect(
          borderRadius: BorderRadius.circular(14),
          child: BackdropFilter(
            filter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
            child: Container(
              decoration: BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [
                    const Color(0xFFFFFFFF).withValues(alpha: 0.80),
                    const Color(0xFFF9F9FB).withValues(alpha: 0.70),
                  ],
                ),
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: const Color(0xFFFFFFFF).withValues(alpha: 0.5), width: 0.5),
                boxShadow: [
                  BoxShadow(color: const Color(0xFF000000).withValues(alpha: 0.04), blurRadius: 14, offset: const Offset(0, 3)),
                ],
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

  const _GlassField({required this.label, required this.controller, this.placeholder = '', this.obscure = false});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 100, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
          Expanded(
            child: CupertinoTextField(
              controller: controller,
              placeholder: placeholder,
              obscureText: obscure,
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              style: const TextStyle(fontSize: 13),
              decoration: BoxDecoration(
                color: const Color(0xFF000000).withValues(alpha: 0.03),
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: const Color(0xFF000000).withValues(alpha: 0.06)),
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
      child: CupertinoTextField(
        controller: controller,
        placeholder: placeholder,
        maxLines: 6,
        minLines: 3,
        padding: const EdgeInsets.all(12),
        style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
        decoration: BoxDecoration(
          color: const Color(0xFF000000).withValues(alpha: 0.03),
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: const Color(0xFF000000).withValues(alpha: 0.06)),
        ),
      ),
    );
  }
}

class _TermChip extends StatelessWidget {
  final String text;
  final VoidCallback onDelete;

  const _TermChip({required this.text, required this.onDelete});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: const Color(0xFF007AFF).withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: const Color(0xFF007AFF).withValues(alpha: 0.2)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(text, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, color: Color(0xFF007AFF))),
          const SizedBox(width: 4),
          GestureDetector(
            onTap: onDelete,
            child: Icon(CupertinoIcons.xmark_circle_fill, size: 14, color: const Color(0xFF007AFF).withValues(alpha: 0.6)),
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
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 11),
      child: Row(
        children: [
          Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C))),
          const Spacer(),
          Container(
            width: 8, height: 8,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: online ? const Color(0xFF34C759) : const Color(0xFFFF3B30),
              boxShadow: [
                BoxShadow(
                  color: (online ? const Color(0xFF34C759) : const Color(0xFFFF3B30)).withValues(alpha: 0.35),
                  blurRadius: 6,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            online ? (detail ?? '已连接') : '未连接',
            style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: online ? const Color(0xFF34C759) : const Color(0xFFFF3B30)),
          ),
        ],
      ),
    );
  }
}
