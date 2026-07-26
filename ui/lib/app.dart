import 'package:flutter/cupertino.dart';

import 'pages/home_page.dart';

class VdubApp extends StatelessWidget {
  const VdubApp({super.key});

  @override
  Widget build(BuildContext context) {
    return const CupertinoApp(
      title: 'vdub',
      debugShowCheckedModeBanner: false,
      theme: CupertinoThemeData(
        brightness: Brightness.light,
        primaryColor: CupertinoColors.systemBlue,
        scaffoldBackgroundColor: CupertinoColors.systemGroupedBackground,
        barBackgroundColor: Color(0xF0F9F9F9),
        textTheme: CupertinoTextThemeData(
          primaryColor: CupertinoColors.systemBlue,
        ),
      ),
      home: HomePage(),
    );
  }
}
