import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../services/api_client.dart';

/// 首次启动引导向导
/// 引导用户完成：Whisper 选择 → LLM 配置 → TTS 配置 → 代理设置
class OnboardingPage extends StatefulWidget {
  final VoidCallback onDone;
  const OnboardingPage({super.key, required this.onDone});

  @override
  State<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends State<OnboardingPage> {
  int _step = 0;
  static const _totalSteps = 4;
  bool _saving = false;

  // Whisper
  String _whisperMode = 'ffmpeg';

  // LLM
  final _llmUrl = TextEditingController(text: 'http://localhost:11434/v1');
  final _llmKey = TextEditingController();
  final _llmModel = TextEditingController(text: 'qwen2.5:7b');

  // TTS
  String _ttsType = 'edge';
  final _ttsVoice = TextEditingController(text: 'zh-CN-YunjianNeural');

  // Proxy
  final _ghProxy = TextEditingController(text: 'https://gh-proxy.com');
  final _hfProxy = TextEditingController(text: 'https://hf-mirror.com');

  @override
  void dispose() {
    _llmUrl.dispose();
    _llmKey.dispose();
    _llmModel.dispose();
    _ttsVoice.dispose();
    _ghProxy.dispose();
    _hfProxy.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return CupertinoPageScaffold(
      backgroundColor: const Color(0xFFF5F5F7),
      child: SafeArea(
        child: Column(
          children: [
            const SizedBox(height: 40),
            const Text('欢迎使用 vdub', style: TextStyle(fontSize: 24, fontWeight: FontWeight.w700, color: Color(0xFF1D1D1F), letterSpacing: -0.5)),
            const SizedBox(height: 6),
            const Text('视频翻译配音工具', style: TextStyle(fontSize: 14, color: Color(0xFF8E8E93))),
            const SizedBox(height: 32),
            _buildStepIndicator(),
            const SizedBox(height: 24),
            Expanded(child: _buildStepContent()),
            _buildNavButtons(),
            const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }

  Widget _buildStepIndicator() {
    const labels = ['语音识别', 'LLM 翻译', '语音合成', '下载代理'];
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: List.generate(_totalSteps, (i) {
        final active = i == _step;
        final done = i < _step;
        return Row(
          children: [
            if (i > 0) Container(width: 32, height: 1, color: done ? CupertinoColors.systemBlue : const Color(0xFFD1D1D6)),
            Column(
              children: [
                Container(
                  width: 28, height: 28,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: done
                        ? CupertinoColors.systemBlue
                        : active
                            ? CupertinoColors.systemBlue.withValues(alpha: 0.15)
                            : const Color(0xFFE5E5EA),
                  ),
                  child: Center(child: done
                      ? const Icon(CupertinoIcons.checkmark, size: 14, color: CupertinoColors.white)
                      : Text('${i + 1}', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600,
                          color: active ? CupertinoColors.systemBlue : const Color(0xFF8E8E93)))),
                ),
                const SizedBox(height: 4),
                Text(labels[i], style: TextStyle(fontSize: 10,
                  color: active ? CupertinoColors.systemBlue : const Color(0xFF8E8E93),
                  fontWeight: active ? FontWeight.w600 : FontWeight.w400)),
              ],
            ),
          ],
        );
      }),
    );
  }

  Widget _buildStepContent() {
    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 200),
      child: [_stepWhisper, _stepLLM, _stepTTS, _stepProxy][_step](),
    );
  }

  Widget _stepWhisper() {
    return _StepCard(
      key: const ValueKey(0),
      title: '语音识别引擎',
      desc: '选择用于语音转文字的引擎',
      child: Column(
        children: [
          _optionTile('FFmpeg 内置 Whisper', '无需额外模型，开箱即用', _whisperMode == 'ffmpeg',
            () => setState(() => _whisperMode = 'ffmpeg')),
          _optionTile('whisper.cpp', '独立引擎，需下载模型文件', _whisperMode == 'whisper-cpp',
            () => setState(() => _whisperMode = 'whisper-cpp')),
        ],
      ),
    );
  }

  Widget _stepLLM() {
    return _StepCard(
      key: const ValueKey(1),
      title: 'LLM 翻译服务',
      desc: '配置用于字幕翻译的大语言模型',
      child: Column(
        children: [
          _inputRow('API 地址', _llmUrl, 'http://localhost:11434/v1'),
          _inputRow('API 密钥', _llmKey, 'sk-xxx (可选)', obscure: true),
          _inputRow('模型名称', _llmModel, 'qwen2.5:7b'),
        ],
      ),
    );
  }

  Widget _stepTTS() {
    return _StepCard(
      key: const ValueKey(2),
      title: '语音合成',
      desc: '选择配音使用的 TTS 服务',
      child: Column(
        children: [
          _optionTile('Edge TTS', '微软免费 TTS，质量好，无需 API 密钥', _ttsType == 'edge',
            () => setState(() => _ttsType = 'edge')),
          _optionTile('OpenAI TTS', '需要 OpenAI API 密钥', _ttsType == 'openai',
            () => setState(() => _ttsType = 'openai')),
          const SizedBox(height: 12),
          _inputRow('语音名称', _ttsVoice, 'zh-CN-YunjianNeural'),
        ],
      ),
    );
  }

  Widget _stepProxy() {
    return _StepCard(
      key: const ValueKey(3),
      title: '下载代理',
      desc: '配置模型和资源下载代理（国内用户建议开启）',
      child: Column(
        children: [
          _inputRow('GitHub 代理', _ghProxy, 'https://gh-proxy.com'),
          _inputRow('HuggingFace 镜像', _hfProxy, 'https://hf-mirror.com'),
        ],
      ),
    );
  }

  Widget _optionTile(String title, String subtitle, bool selected, VoidCallback onTap) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: selected ? CupertinoColors.systemBlue.withValues(alpha: 0.08) : const Color(0xFFF5F5F7),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: selected ? CupertinoColors.systemBlue.withValues(alpha: 0.3) : const Color(0xFFE5E5EA)),
        ),
        child: Row(
          children: [
            Expanded(child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: TextStyle(fontSize: 14, fontWeight: FontWeight.w500,
                  color: selected ? CupertinoColors.systemBlue : const Color(0xFF1D1D1F))),
                Text(subtitle, style: const TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
              ],
            )),
            if (selected) const Icon(CupertinoIcons.checkmark_circle_fill, size: 20, color: CupertinoColors.systemBlue),
          ],
        ),
      ),
    );
  }

  Widget _inputRow(String label, TextEditingController ctrl, String placeholder, {bool obscure = false}) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: const TextStyle(fontSize: 12, color: Color(0xFF8E8E93))),
          const SizedBox(height: 4),
          CupertinoTextField(
            controller: ctrl,
            placeholder: placeholder,
            obscureText: obscure,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            style: const TextStyle(fontSize: 13),
            decoration: BoxDecoration(
              color: const Color(0xFFF5F5F7),
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: const Color(0xFFE5E5EA)),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildNavButtons() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 40),
      child: Row(
        children: [
          if (_step > 0)
            CupertinoButton(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
              onPressed: () => setState(() => _step--),
              child: const Text('上一步', style: TextStyle(fontSize: 14)),
            ),
          const Spacer(),
          if (_step < _totalSteps - 1)
            CupertinoButton.filled(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
              borderRadius: BorderRadius.circular(10),
              onPressed: () => setState(() => _step++),
              child: const Text('下一步', style: TextStyle(fontSize: 14, color: CupertinoColors.white)),
            )
          else
            CupertinoButton.filled(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
              borderRadius: BorderRadius.circular(10),
              onPressed: _saving ? null : _finish,
              child: Text(_saving ? '保存中...' : '完成设置', style: const TextStyle(fontSize: 14, color: CupertinoColors.white)),
            ),
        ],
      ),
    );
  }

  Future<void> _finish() async {
    setState(() => _saving = true);
    try {
      final api = context.read<ApiClient>();
      await api.updateConfig({
        'pipeline': {
          'whisper_mode': _whisperMode,
        },
        'llm': {
          'base_url': _llmUrl.text.trim(),
          'api_key': _llmKey.text.trim(),
          'model': _llmModel.text.trim(),
        },
        'tts': {
          'type': _ttsType,
          'voice': _ttsVoice.text.trim(),
        },
      });

      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool('onboarding_done', true);
      await prefs.setString('gh_proxy', _ghProxy.text.trim());
      await prefs.setString('hf_proxy', _hfProxy.text.trim());

      widget.onDone();
    } catch (e) {
      if (mounted) {
        showCupertinoDialog(context: context, builder: (_) => CupertinoAlertDialog(
          content: Text('保存失败: $e'),
          actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))],
        ));
      }
    }
    if (mounted) setState(() => _saving = false);
  }
}

class _StepCard extends StatelessWidget {
  final String title;
  final String desc;
  final Widget child;

  const _StepCard({super.key, required this.title, required this.desc, required this.child});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 40),
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: CupertinoColors.white,
        borderRadius: BorderRadius.circular(14),
        boxShadow: const [BoxShadow(color: Color(0x0A000000), blurRadius: 12, offset: Offset(0, 4))],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(title, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F))),
          const SizedBox(height: 4),
          Text(desc, style: const TextStyle(fontSize: 13, color: Color(0xFF8E8E93))),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }
}
