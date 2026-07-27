import 'package:flutter/cupertino.dart';
import 'package:flutter/services.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'ui/features/home/views/home_page.dart';
import 'ui/features/onboarding/views/onboarding_page.dart';

class VdubApp extends ConsumerWidget {
  const VdubApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return CupertinoApp(
      title: 'vdub',
      debugShowCheckedModeBanner: false,
      theme: const CupertinoThemeData(
        primaryColor: CupertinoColors.systemBlue,
        scaffoldBackgroundColor: Color(0xFFF5F5F7),
        barBackgroundColor: Color(0xF0F9F9F9),
        textTheme: CupertinoTextThemeData(
          primaryColor: CupertinoColors.systemBlue,
        ),
      ),
      home: const _AppGate(),
    );
  }
}

class _AppGate extends StatefulWidget {
  const _AppGate();

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

  /// 标题栏样式跟随系统深浅模式
  void _updateSystemOverlay() {
    final brightness =
        WidgetsBinding.instance.platformDispatcher.platformBrightness;
    SystemChrome.setSystemUIOverlayStyle(
      brightness == Brightness.dark
          ? SystemUiOverlayStyle.light.copyWith(
              statusBarColor: const Color(0xFF1C1C1E),
              systemNavigationBarColor: const Color(0xFF1C1C1E),
            )
          : SystemUiOverlayStyle.dark.copyWith(
              statusBarColor: const Color(0xFFF5F5F7),
              systemNavigationBarColor: const Color(0xFFF5F5F7),
            ),
    );
  }

  Future<void> _checkOnboarding() async {
    final prefs = await SharedPreferences.getInstance();
    setState(
        () => _onboardingDone = prefs.getBool('onboarding_done') ?? false);
  }

  @override
  Widget build(BuildContext context) {
    if (_onboardingDone == null) {
      return const CupertinoPageScaffold(
        child: Center(child: CupertinoActivityIndicator(radius: 14)),
      );
    }

    if (!_onboardingDone!) {
      return OnboardingPage(
          onDone: () => setState(() => _onboardingDone = true));
    }

    return const HomePage();
  }
}
