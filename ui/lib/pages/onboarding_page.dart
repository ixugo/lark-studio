import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:shared_preferences/shared_preferences.dart';

class OnboardingPage extends StatefulWidget {
  final VoidCallback onDone;
  const OnboardingPage({super.key, required this.onDone});

  @override
  State<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends State<OnboardingPage> {
  String _whisperMode = 'ffmpeg';

  @override
  Widget build(BuildContext context) {
    return CupertinoPageScaffold(
      backgroundColor: CupertinoColors.transparent,
      child: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [Color(0xFFF8F8FA), Color(0xFFF0F0F5), Color(0xFFECECF4)],
          ),
        ),
        child: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 48, vertical: 40),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Container(
                    width: 64, height: 64,
                    decoration: BoxDecoration(
                      gradient: const LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [Color(0xFF007AFF), Color(0xFF5AC8FA)],
                      ),
                      borderRadius: BorderRadius.circular(18),
                      boxShadow: [
                        BoxShadow(color: const Color(0xFF007AFF).withValues(alpha: 0.25), blurRadius: 20, offset: const Offset(0, 6)),
                      ],
                    ),
                    child: const Center(
                      child: Text('v', style: TextStyle(fontSize: 32, fontWeight: FontWeight.w800, color: CupertinoColors.white, letterSpacing: -1)),
                    ),
                  ),
                  const SizedBox(height: 20),
                  const Text('vdub', style: TextStyle(fontSize: 32, fontWeight: FontWeight.w800, color: Color(0xFF1D1D1F), letterSpacing: -1.2)),
                  const SizedBox(height: 4),
                  const Text('视频翻译配音工具', style: TextStyle(fontSize: 14, color: Color(0xFF8E8E93), letterSpacing: 0.2)),
                  const SizedBox(height: 44),
                  _glassCard(),
                  const SizedBox(height: 36),
                  SizedBox(
                    width: double.infinity,
                    child: CupertinoButton.filled(
                      borderRadius: BorderRadius.circular(14),
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      onPressed: _finish,
                      child: const Text('开始使用', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600, color: CupertinoColors.white)),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _glassCard() {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          padding: const EdgeInsets.all(24),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [
                const Color(0xFFFFFFFF).withValues(alpha: 0.82),
                const Color(0xFFF9F9FB).withValues(alpha: 0.72),
              ],
            ),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: const Color(0xFFFFFFFF).withValues(alpha: 0.5), width: 0.5),
            boxShadow: [
              BoxShadow(color: const Color(0xFF000000).withValues(alpha: 0.04), blurRadius: 16, offset: const Offset(0, 4)),
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('选择语音识别引擎', style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F), letterSpacing: -0.3)),
              const SizedBox(height: 4),
              const Text('其余配置可在设置页随时修改', style: TextStyle(fontSize: 13, color: Color(0xFF8E8E93))),
              const SizedBox(height: 18),
              _optionTile(
                'FFmpeg 内置 Whisper',
                '无需额外模型，开箱即用',
                _whisperMode == 'ffmpeg',
                () => setState(() => _whisperMode = 'ffmpeg'),
              ),
              _optionTile(
                'whisper.cpp',
                '独立引擎，需下载模型文件，识别精度更高',
                _whisperMode == 'whisper-cpp',
                () => setState(() => _whisperMode = 'whisper-cpp'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _optionTile(String title, String subtitle, bool selected, VoidCallback onTap) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 180),
        margin: const EdgeInsets.symmetric(vertical: 5),
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: selected
              ? const Color(0xFF007AFF).withValues(alpha: 0.08)
              : const Color(0xFF000000).withValues(alpha: 0.02),
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: selected
                ? const Color(0xFF007AFF).withValues(alpha: 0.30)
                : const Color(0xFF000000).withValues(alpha: 0.06),
          ),
        ),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: TextStyle(
                    fontSize: 14, fontWeight: FontWeight.w500,
                    color: selected ? const Color(0xFF007AFF) : const Color(0xFF1D1D1F),
                  )),
                  const SizedBox(height: 2),
                  Text(subtitle, style: const TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
                ],
              ),
            ),
            if (selected)
              const Icon(CupertinoIcons.checkmark_circle_fill, size: 20, color: Color(0xFF007AFF)),
          ],
        ),
      ),
    );
  }

  Future<void> _finish() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('whisper_mode', _whisperMode);
    await prefs.setBool('onboarding_done', true);
    widget.onDone();
  }
}
