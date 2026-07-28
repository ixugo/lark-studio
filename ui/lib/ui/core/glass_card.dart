import 'dart:ui';

import 'package:flutter/widgets.dart';

import 'app_colors.dart';

/// Liquid Glass 玻璃卡片：半透明底 + 镜面细描边 + 顶部高光棱线。
/// 传入 onTap 即获得 hover 提亮与按压回弹，作可点卡片用。
class GlassCard extends StatefulWidget {
  final Widget child;
  final EdgeInsetsGeometry? padding;
  final double radius;
  final VoidCallback? onTap;

  const GlassCard({
    super.key,
    required this.child,
    this.padding,
    this.radius = 13,
    this.onTap,
  });

  @override
  State<GlassCard> createState() => _GlassCardState();
}

class _GlassCardState extends State<GlassCard> {
  bool _hovering = false;
  bool _pressing = false;

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final tappable = widget.onTap != null;
    final base = _hovering && tappable ? c.glassCardHover : c.glassCardBg;
    // 顶部 12% 渐亮模拟玻璃受光棱线（Flutter 的 Border 四边异色时禁止圆角，故用渐变实现）
    final sheen = Color.lerp(
      base,
      const Color(0xFFFFFFFF),
      c.isDark ? 0.10 : 0.45,
    )!;

    Widget card = ClipRRect(
      borderRadius: BorderRadius.circular(widget.radius),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 24, sigmaY: 24),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 140),
          curve: Curves.easeOut,
          padding: widget.padding,
          transform: _pressing
              ? (Matrix4.identity()..scaleByDouble(0.985, 0.985, 1.0, 1.0))
              : Matrix4.identity(),
          transformAlignment: Alignment.center,
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              stops: const [0.0, 0.12, 1.0],
              colors: [sheen, base, base],
            ),
            borderRadius: BorderRadius.circular(widget.radius),
            border: Border.all(color: c.glassStroke, width: 0.5),
            boxShadow: [
              BoxShadow(
                color: const Color(
                  0xFF000000,
                ).withValues(alpha: c.isDark ? 0.22 : 0.06),
                blurRadius: 12,
                offset: const Offset(0, 2),
              ),
            ],
          ),
          child: widget.child,
        ),
      ),
    );

    if (!tappable) return card;
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      onEnter: (_) => setState(() => _hovering = true),
      onExit: (_) => setState(() => _hovering = false),
      child: GestureDetector(
        onTap: widget.onTap,
        onTapDown: (_) => setState(() => _pressing = true),
        onTapUp: (_) => setState(() => _pressing = false),
        onTapCancel: () => setState(() => _pressing = false),
        child: card,
      ),
    );
  }
}

/// 玻璃横条：用于侧栏、顶栏、状态栏等贴边区域，只模糊不上浮起阴影。
class GlassBar extends StatelessWidget {
  final Widget child;
  final double? height;
  final EdgeInsetsGeometry? padding;
  final Border? border;

  const GlassBar({
    super.key,
    required this.child,
    this.height,
    this.padding,
    this.border,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          height: height,
          padding: padding,
          decoration: BoxDecoration(color: c.glassBarBg, border: border),
          child: child,
        ),
      ),
    );
  }
}
