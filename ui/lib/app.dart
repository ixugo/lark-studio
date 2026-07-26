import 'package:flutter/cupertino.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'pages/home_page.dart';
import 'pages/onboarding_page.dart';

class VdubApp extends StatelessWidget {
  const VdubApp({super.key});

  @override
  Widget build(BuildContext context) {
    return CupertinoApp(
      title: 'vdub',
      debugShowCheckedModeBanner: false,
      theme: const CupertinoThemeData(
        brightness: Brightness.light,
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

class _AppGateState extends State<_AppGate> {
  bool? _onboardingDone;

  @override
  void initState() {
    super.initState();
    _checkOnboarding();
  }

  Future<void> _checkOnboarding() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() => _onboardingDone = prefs.getBool('onboarding_done') ?? false);
  }

  @override
  Widget build(BuildContext context) {
    if (_onboardingDone == null) {
      return const CupertinoPageScaffold(
        child: Center(child: CupertinoActivityIndicator(radius: 14)),
      );
    }

    if (!_onboardingDone!) {
      return OnboardingPage(onDone: () => setState(() => _onboardingDone = true));
    }

    return const HomePage();
  }
}
