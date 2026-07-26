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

  @override
  void initState() {
    super.initState();
    _loadConfig();
  }

  @override
  void dispose() {
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
      final cfg = await context.read<ApiClient>().getConfig();
      _applyConfig(cfg);
      setState(() { _config = cfg; _loading = false; });
    } catch (e) {
      setState(() { _error = '$e'; _loading = false; });
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
        },
      };
      final cfg = await context.read<ApiClient>().updateConfig(updates);
      _applyConfig(cfg);
      setState(() { _config = cfg; _saving = false; });
      if (mounted) {
        _showToast('配置已保存');
      }
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
        _toolbar(),
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

  Widget _toolbar() {
    return Container(
      height: 52,
      padding: const EdgeInsets.symmetric(horizontal: 20),
      decoration: const BoxDecoration(
        color: Color(0xFFFAFAFA),
        border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5)),
      ),
      child: Row(
        children: [
          const Text('设置', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
          const Spacer(),
          if (_config != null)
            CupertinoButton.filled(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
              borderRadius: BorderRadius.circular(8),
              onPressed: _saving ? null : _saveConfig,
              child: Text(_saving ? '保存中...' : '保存配置', style: const TextStyle(fontSize: 13, color: CupertinoColors.white)),
            ),
        ],
      ),
    );
  }

  Widget _buildForm(BackendService backend, WebSocketService wsService) {
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        _section('连接状态', [
          _statusRow('后端服务', backend.online),
          _statusRow('WebSocket', wsService.connected),
        ]),
        const SizedBox(height: 20),
        _section('LLM 翻译配置', [
          _field('API 地址', _llmBaseUrl, placeholder: 'http://localhost:11434/v1'),
          _field('API 密钥', _llmApiKey, placeholder: 'sk-xxx', obscure: true),
          _field('模型名称', _llmModel, placeholder: 'qwen2.5:7b'),
        ]),
        const SizedBox(height: 20),
        _section('TTS 语音合成', [
          _segmentRow('TTS 类型', {'edge': 'Edge TTS', 'openai': 'OpenAI TTS'}, _ttsType, (v) => setState(() => _ttsType = v)),
          _field('语音名称', _ttsVoice, placeholder: 'zh-CN-YunjianNeural'),
          if (_ttsType == 'openai') ...[
            _field('API 地址', _ttsBaseUrl, placeholder: 'https://api.openai.com/v1'),
            _field('API 密钥', _ttsApiKey, placeholder: 'sk-xxx', obscure: true),
            _field('模型', _ttsModel, placeholder: 'tts-1'),
          ],
        ]),
        const SizedBox(height: 20),
        _section('流水线配置', [
          _segmentRow('Whisper 模式', {'ffmpeg': 'FFmpeg', 'whisper-cpp': 'whisper.cpp'}, _whisperMode, (v) => setState(() => _whisperMode = v)),
          _field('Whisper 模型路径', _whisperModel, placeholder: '/path/to/ggml-large-v3.bin'),
          _field('FFmpeg 路径', _ffmpegBin, placeholder: '留空使用 PATH'),
          _field('默认目标语言', _targetLang, placeholder: 'zh-CN'),
          _sliderRow('Worker 数量', _workers, 1, 4, (v) => setState(() => _workers = v)),
        ]),
        const SizedBox(height: 20),
        _section('翻译提示词', [
          _multiLineField(
            '自定义系统提示词',
            _translatePrompt,
            placeholder: '留空使用内置默认模板。\n可用变量: {{target_lang}} {{count}}',
          ),
        ]),
        const SizedBox(height: 40),
      ],
    );
  }

  Widget _section(String title, List<Widget> children) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(left: 4, bottom: 8),
          child: Text(title, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: Color(0xFF8E8E93))),
        ),
        Container(
          decoration: BoxDecoration(
            color: CupertinoColors.white,
            borderRadius: BorderRadius.circular(10),
            boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 8, offset: Offset(0, 2))],
          ),
          child: Column(children: children),
        ),
      ],
    );
  }

  Widget _multiLineField(String label, TextEditingController ctrl, {String placeholder = ''}) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C))),
          const SizedBox(height: 8),
          CupertinoTextField(
            controller: ctrl,
            placeholder: placeholder,
            maxLines: 6,
            minLines: 3,
            padding: const EdgeInsets.all(10),
            style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
            decoration: BoxDecoration(
              color: const Color(0xFFF5F5F7),
              borderRadius: BorderRadius.circular(6),
            ),
          ),
        ],
      ),
    );
  }

  Widget _field(String label, TextEditingController ctrl, {String placeholder = '', bool obscure = false}) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 120, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
          Expanded(
            child: CupertinoTextField(
              controller: ctrl,
              placeholder: placeholder,
              obscureText: obscure,
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              style: const TextStyle(fontSize: 13),
              decoration: BoxDecoration(
                color: const Color(0xFFF5F5F7),
                borderRadius: BorderRadius.circular(6),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _statusRow(String label, bool online) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C))),
          const Spacer(),
          Container(width: 8, height: 8, decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: online ? CupertinoColors.systemGreen : CupertinoColors.systemRed,
          )),
          const SizedBox(width: 6),
          Text(online ? '已连接' : '未连接', style: TextStyle(
            fontSize: 13, color: online ? CupertinoColors.systemGreen : CupertinoColors.systemRed,
          )),
        ],
      ),
    );
  }

  Widget _segmentRow(String label, Map<String, String> options, String value, ValueChanged<String> onChanged) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 120, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
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

  Widget _sliderRow(String label, int value, int min, int max, ValueChanged<int> onChanged) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          SizedBox(width: 120, child: Text(label, style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
          Expanded(
            child: CupertinoSlider(
              value: value.toDouble(),
              min: min.toDouble(),
              max: max.toDouble(),
              divisions: max - min,
              onChanged: (v) => onChanged(v.round()),
            ),
          ),
          SizedBox(width: 30, child: Text('$value', style: const TextStyle(fontSize: 13, color: Color(0xFF3A3A3C)))),
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
