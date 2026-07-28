import 'package:flutter/widgets.dart';
import 'app_colors.dart';

/// 统一按钮样式（Liquid Glass 胶囊形）
/// primary: 强调色填充，带顶部高光渐变
/// secondary: 玻璃底 + 细描边
/// 所有按钮保证：最小高度 32、水平内边距 16、文字居中、胶囊圆角
enum AppButtonStyle { primary, secondary }

class AppButton extends StatefulWidget {
  final Widget child;
  final VoidCallback? onPressed;
  final Color? color;
  final AppButtonStyle style;
  final double? minWidth;

  const AppButton({
    super.key,
    required this.child,
    this.onPressed,
    this.color,
    this.style = AppButtonStyle.primary,
    this.minWidth,
  });

  const AppButton.secondary({
    super.key,
    required this.child,
    this.onPressed,
    this.color,
    this.minWidth,
  }) : style = AppButtonStyle.secondary;

  @override
  State<AppButton> createState() => _AppButtonState();
}

class _AppButtonState extends State<AppButton> {
  bool _pressing = false;
  bool _hovering = false;

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final enabled = widget.onPressed != null;
    final isPrimary = widget.style == AppButtonStyle.primary;
    final base = widget.color ?? c.accent;

    // 主按钮用纵向渐变模拟玻璃受光，hover 时整体提亮
    final primaryDecoration = BoxDecoration(
      gradient: LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [
          Color.lerp(
            base,
            const Color(0xFFFFFFFF),
            _hovering && enabled ? 0.34 : 0.22,
          )!,
          Color.lerp(
            base,
            const Color(0xFFFFFFFF),
            _hovering && enabled ? 0.08 : 0.0,
          )!,
        ],
      ),
      borderRadius: BorderRadius.circular(999),
      border: Border.all(color: c.glassTopLine, width: 0.5),
    );

    final secondaryBase = widget.color;
    final secondaryDecoration = BoxDecoration(
      color: secondaryBase == null
          ? (_hovering && enabled ? c.glassCardHover : c.glassCardBg)
          : secondaryBase.withValues(alpha: _hovering && enabled ? 0.16 : 0.09),
      borderRadius: BorderRadius.circular(999),
      border: Border.all(
        color: secondaryBase?.withValues(alpha: 0.28) ?? c.glassStroke,
        width: 0.5,
      ),
    );

    return MouseRegion(
      cursor: enabled ? SystemMouseCursors.click : SystemMouseCursors.basic,
      onEnter: (_) => setState(() => _hovering = true),
      onExit: (_) => setState(() => _hovering = false),
      child: GestureDetector(
        onTapDown: enabled ? (_) => setState(() => _pressing = true) : null,
        onTapUp: enabled
            ? (_) {
                setState(() => _pressing = false);
                widget.onPressed?.call();
              }
            : null,
        onTapCancel: () => setState(() => _pressing = false),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 100),
          constraints: BoxConstraints(
            minHeight: 32,
            minWidth: widget.minWidth ?? 60,
          ),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
          transform: _pressing
              ? (Matrix4.identity()..scaleByDouble(0.97, 0.97, 1.0, 1.0))
              : Matrix4.identity(),
          transformAlignment: Alignment.center,
          decoration: isPrimary ? primaryDecoration : secondaryDecoration,
          foregroundDecoration: !enabled
              ? BoxDecoration(
                  color: c.bg.withValues(alpha: 0.45),
                  borderRadius: BorderRadius.circular(999),
                )
              : null,
          child: DefaultTextStyle.merge(
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: isPrimary
                  ? const Color(0xFFFFFFFF)
                  : secondaryBase ?? c.textPrimary,
            ),
            child: IconTheme.merge(
              data: IconThemeData(
                size: 14,
                color: isPrimary
                    ? const Color(0xFFFFFFFF)
                    : secondaryBase ?? c.textSecondary,
              ),
              child: Center(
                widthFactor: 1,
                heightFactor: 1,
                child: widget.child,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
