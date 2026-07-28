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

  // ── 背景层级（半透明，令环境色透出，成玻璃之质） ──
  Color get bg => isDark ? const Color(0xFF000000) : const Color(0xFFF5F5F7);
  Color get contentBg =>
      isDark ? const Color(0xF2141418) : const Color(0xFAFFFFFF);
  Color get cardBg =>
      isDark ? const Color(0x8C2C2C32) : const Color(0xB8FFFFFF);
  Color get cardBgHover =>
      isDark ? const Color(0x9E3A3A42) : const Color(0xD9FFFFFF);
  Color get barBg => isDark ? const Color(0x8C17171B) : const Color(0x99FAFAFC);
  Color get inputBg =>
      isDark ? const Color(0x12FFFFFF) : const Color(0x0C000000);
  Color get surfaceBg =>
      isDark ? const Color(0x802C2C32) : const Color(0x80FFFFFF);

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

  /// 主题强调色：暗色下用更亮的系统蓝保证可读性
  Color get accent => isDark ? const Color(0xFF0A84FF) : blue;

  // ── 玻璃材质（Liquid Glass） ──
  // 卡片底：半透明让背景色微微透出
  Color get glassCardBg =>
      isDark ? const Color(0x8C2C2C32) : const Color(0x9EFFFFFF);
  Color get glassCardHover =>
      isDark ? const Color(0x9E3A3A42) : const Color(0xD1FFFFFF);
  // 横条底（侧栏/顶栏/状态栏）：比卡片更暗一档
  Color get glassBarBg =>
      isDark ? const Color(0x8C17171B) : const Color(0x99FAFAFC);
  // 镜面细描边
  Color get glassStroke =>
      isDark ? const Color(0x1AFFFFFF) : const Color(0x17000000);
  // 顶部高光描边，模拟玻璃受光棱线
  Color get glassTopLine =>
      isDark ? const Color(0x24FFFFFF) : const Color(0xD9FFFFFF);
}
