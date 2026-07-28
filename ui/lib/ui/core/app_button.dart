import 'package:flutter/widgets.dart';
import 'app_colors.dart';

/// 统一按钮样式
/// primary: 蓝色/绿色等填充按钮
/// secondary: 带边框无填充按钮
/// 所有按钮保证：最小高度 32、水平内边距 16、文字居中、圆角 8
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

    final bgColor = !enabled
        ? (isPrimary ? (widget.color ?? AppColors.blue).withValues(alpha: 0.4) : c.inputBg)
        : isPrimary
            ? (widget.color ?? AppColors.blue)
            : _hovering
                ? c.cardBgHover
                : const Color(0x00000000);

    final borderColor = isPrimary
        ? const Color(0x00000000)
        : c.borderLight;

    return MouseRegion(
      cursor: enabled ? SystemMouseCursors.click : SystemMouseCursors.basic,
      onEnter: (_) => setState(() => _hovering = true),
      onExit: (_) => setState(() => _hovering = false),
      child: GestureDetector(
        onTapDown: enabled ? (_) => setState(() => _pressing = true) : null,
        onTapUp: enabled ? (_) { setState(() => _pressing = false); widget.onPressed?.call(); } : null,
        onTapCancel: () => setState(() => _pressing = false),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 100),
          constraints: BoxConstraints(
            minHeight: 32,
            minWidth: widget.minWidth ?? 60,
          ),
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
          transform: _pressing ? (Matrix4.identity()..scale(0.97, 0.97)) : Matrix4.identity(),
          transformAlignment: Alignment.center,
          decoration: BoxDecoration(
            color: bgColor,
            borderRadius: BorderRadius.circular(8),
            border: isPrimary ? null : Border.all(color: borderColor, width: 0.5),
          ),
          child: DefaultTextStyle.merge(
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: isPrimary
                  ? const Color(0xFFFFFFFF)
                  : c.textPrimary,
            ),
            child: IconTheme.merge(
              data: IconThemeData(
                size: 14,
                color: isPrimary
                    ? const Color(0xFFFFFFFF)
                    : c.textSecondary,
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
