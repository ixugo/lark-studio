import 'dart:ui';
import 'package:flutter/cupertino.dart';

/// 液态玻璃卡片——全局共享 UI 组件
class GlassCard extends StatelessWidget {
  final Widget child;
  const GlassCard({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
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
            border: Border.all(
              color: const Color(0xFFFFFFFF).withValues(alpha: 0.5),
              width: 0.5,
            ),
            boxShadow: [
              BoxShadow(
                  color: const Color(0xFF000000).withValues(alpha: 0.04),
                  blurRadius: 16,
                  offset: const Offset(0, 4)),
              BoxShadow(
                  color: const Color(0xFF000000).withValues(alpha: 0.02),
                  blurRadius: 4,
                  offset: const Offset(0, 1)),
            ],
          ),
          child: child,
        ),
      ),
    );
  }
}
