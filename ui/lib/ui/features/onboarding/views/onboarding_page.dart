import 'package:flutter/cupertino.dart'
    show CupertinoColors, CupertinoIcons, CupertinoPageScaffold;
import 'package:flutter/widgets.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../../../core/app_colors.dart';
import '../../../core/glass_card.dart';

class OnboardingPage extends StatefulWidget {
  final VoidCallback onDone;
  const OnboardingPage({super.key, required this.onDone});

  @override
  State<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends State<OnboardingPage> {
  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return CupertinoPageScaffold(
      backgroundColor: CupertinoColors.transparent,
      child: Container(
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: c.isDark
                ? const [
                    Color(0xFF0B0B12),
                    Color(0xFF12121C),
                    Color(0xFF161624),
                  ]
                : const [
                    Color(0xFFF8F8FA),
                    Color(0xFFF0F0F5),
                    Color(0xFFECECF4),
                  ],
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
                    width: 64,
                    height: 64,
                    decoration: BoxDecoration(
                      gradient: const LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [Color(0xFF007AFF), Color(0xFF5AC8FA)],
                      ),
                      borderRadius: BorderRadius.circular(18),
                      boxShadow: [
                        BoxShadow(
                          color: const Color(
                            0xFF007AFF,
                          ).withValues(alpha: 0.25),
                          blurRadius: 20,
                          offset: const Offset(0, 6),
                        ),
                      ],
                    ),
                    child: const Center(
                      child: Text(
                        'v',
                        style: TextStyle(
                          fontSize: 32,
                          fontWeight: FontWeight.w800,
                          color: CupertinoColors.white,
                          letterSpacing: -1,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),
                  Text(
                    'vdub',
                    style: TextStyle(
                      fontSize: 32,
                      fontWeight: FontWeight.w800,
                      color: c.textPrimary,
                      letterSpacing: -1.2,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '视频翻译配音工具',
                    style: TextStyle(
                      fontSize: 14,
                      color: c.textTertiary,
                      letterSpacing: 0.2,
                    ),
                  ),
                  const SizedBox(height: 44),
                  _glassCard(c),
                  const SizedBox(height: 36),
                  SizedBox(
                    width: double.infinity,
                    child: PushButton(
                      controlSize: ControlSize.large,
                      color: c.accent,
                      onPressed: _finish,
                      child: const Text(
                        '开始使用',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w600,
                          color: MacosColors.white,
                        ),
                      ),
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

  Widget _glassCard(AppColors c) {
    return GlassCard(
      radius: 16,
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            '语音识别已就绪',
            style: TextStyle(
              fontSize: 17,
              fontWeight: FontWeight.w600,
              color: c.textPrimary,
              letterSpacing: -0.3,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            '使用 whisper.cpp；首次使用时只需下载所选模型',
            style: TextStyle(fontSize: 13, color: c.textTertiary),
          ),
          const SizedBox(height: 18),
          Row(
            children: [
              Icon(CupertinoIcons.waveform, size: 20, color: c.accent),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  '模型可在“引擎”中下载与切换。',
                  style: TextStyle(fontSize: 13, color: c.textSecondary),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Future<void> _finish() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('whisper_mode', 'whisper-cpp');
    await prefs.setBool('onboarding_done', true);
    widget.onDone();
  }
}
