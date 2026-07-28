import 'package:flutter/widgets.dart';
import 'package:macos_ui/macos_ui.dart';

/// 应用颜色 token 系统
/// 亮色：白底深字，高对比度
/// 暗色：黑底亮字，舒适阅读
/// 所有页面通过 AppColors.of(context) 获取，禁止硬编码
class AppColors {
  final bool isDark;
  const AppColors._(this.isDark);

  factory AppColors.of(BuildContext context) {
    final isDark = MacosTheme.brightnessOf(context) == Brightness.dark;
    return AppColors._(isDark);
  }

  // ── 背景层级 ──
  Color get bg => isDark ? const Color(0xFF000000) : const Color(0xFFF5F5F7);
  Color get contentBg =>
      isDark ? const Color(0xFF1C1C1E) : const Color(0xFFFFFFFF);
  Color get cardBg =>
      isDark ? const Color(0xFF2C2C2E) : const Color(0xFFFFFFFF);
  Color get cardBgHover =>
      isDark ? const Color(0xFF3A3A3C) : const Color(0xFFF2F2F7);
  Color get barBg =>
      isDark ? const Color(0xFF1C1C1E) : const Color(0xFFFFFFFF);
  Color get inputBg =>
      isDark ? const Color(0xFF3A3A3C) : const Color(0xFFF2F2F7);
  Color get surfaceBg =>
      isDark ? const Color(0xFF2C2C2E) : const Color(0xFFF9F9F9);

  // ── 边框 ──
  Color get border =>
      isDark ? const Color(0xFF48484A) : const Color(0xFFD1D1D6);
  Color get borderLight =>
      isDark ? const Color(0xFF3A3A3C) : const Color(0xFFE5E5EA);
  Color get borderSubtle =>
      isDark ? const Color(0xFF2C2C2E) : const Color(0xFFE8E8ED);

  // ── 文字（亮色下深色高对比，暗色下亮色舒适） ──
  Color get textPrimary =>
      isDark ? const Color(0xFFF5F5F7) : const Color(0xFF1D1D1F);
  Color get textSecondary =>
      isDark ? const Color(0xFFAEAEB2) : const Color(0xFF3A3A3C);
  Color get textTertiary =>
      isDark ? const Color(0xFF8E8E93) : const Color(0xFF6E6E73);
  Color get textQuaternary =>
      isDark ? const Color(0xFF636366) : const Color(0xFF8E8E93);
  Color get textPlaceholder =>
      isDark ? const Color(0xFF636366) : const Color(0xFFA1A1A6);

  // ── 功能色 ──
  static const blue = Color(0xFF007AFF);
  static const green = Color(0xFF34C759);
  static const red = Color(0xFFFF3B30);
  static const orange = Color(0xFFFF9500);
  static const yellow = Color(0xFFFFCC00);
}
