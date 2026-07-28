import 'package:flutter/material.dart' show ThemeMode;
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'providers.dart';
import 'ui/features/home/views/home_page.dart';
import 'ui/features/onboarding/views/onboarding_page.dart';

class VdubApp extends ConsumerWidget {
  const VdubApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.watch(themeProvider);
    final platformBrightness = MediaQuery.platformBrightnessOf(context);
    final brightness =
        ref.read(themeProvider.notifier).resolve(platformBrightness);
    final isDark = brightness == Brightness.dark;

    return MacosApp(
      title: 'vdub',
      debugShowCheckedModeBanner: false,
      themeMode: isDark ? ThemeMode.dark : ThemeMode.light,
      theme: MacosThemeData.light(),
      darkTheme: MacosThemeData.dark(),
      home: _AppGate(isDark: isDark),
    );
  }
}

class _AppGate extends StatefulWidget {
  final bool isDark;
  const _AppGate({required this.isDark});

  @override
  State<_AppGate> createState() => _AppGateState();
}

class _AppGateState extends State<_AppGate> with WidgetsBindingObserver {
  bool? _onboardingDone;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _checkOnboarding();
    _updateSystemOverlay();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangePlatformBrightness() {
    _updateSystemOverlay();
  }

  static const _windowChannel = MethodChannel('vdub/window');

  @override
  void didUpdateWidget(covariant _AppGate oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.isDark != widget.isDark) _updateSystemOverlay();
  }

  void _updateSystemOverlay() {
    _windowChannel.invokeMethod('setDarkMode', widget.isDark);
    if (widget.isDark) {
      SystemChrome.setSystemUIOverlayStyle(
        SystemUiOverlayStyle.light.copyWith(
          statusBarColor: const Color(0xFF000000),
          systemNavigationBarColor: const Color(0xFF000000),
        ),
      );
    } else {
      SystemChrome.setSystemUIOverlayStyle(
        SystemUiOverlayStyle.dark.copyWith(
          statusBarColor: const Color(0xFFF2F2F7),
          systemNavigationBarColor: const Color(0xFFF2F2F7),
        ),
      );
    }
  }

  Future<void> _checkOnboarding() async {
    final prefs = await SharedPreferences.getInstance();
    setState(
        () => _onboardingDone = prefs.getBool('onboarding_done') ?? false);
  }

  @override
  Widget build(BuildContext context) {
    if (_onboardingDone == null) {
      return const Center(child: ProgressCircle(radius: 14));
    }

    if (!_onboardingDone!) {
      return OnboardingPage(
          onDone: () => setState(() => _onboardingDone = true));
    }

    return const HomePage();
  }
}
