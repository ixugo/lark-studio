import 'dart:ui';
import 'package:flutter/cupertino.dart';

/// 暗色玻璃卡片——全局共享 UI 组件
class GlassCard extends StatelessWidget {
  final Widget child;
  const GlassCard({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
        child: Container(
          decoration: BoxDecoration(
            color: const Color(0xFF1C1C1E).withValues(alpha: 0.85),
            borderRadius: BorderRadius.circular(12),
            border: Border.all(
              color: const Color(0xFFFFFFFF).withValues(alpha: 0.06),
              width: 0.5,
            ),
          ),
          child: child,
        ),
      ),
    );
  }
}
