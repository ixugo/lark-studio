import 'dart:ui';
import 'package:flutter/cupertino.dart' show CupertinoColors, CupertinoIcons;
import 'package:flutter/widgets.dart';

import 'app_colors.dart';

/// 桌面风格下拉选择器，锚定到触发按钮正下方弹出 overlay
/// 参考 Apple Design: 空间一致性 + 锚定到触发源
class DesktopDropdown extends StatefulWidget {
  final String value;
  final List<String> items;
  final ValueChanged<String> onChanged;
  final Widget child;

  const DesktopDropdown({
    super.key,
    required this.value,
    required this.items,
    required this.onChanged,
    required this.child,
  });

  @override
  State<DesktopDropdown> createState() => _DesktopDropdownState();
}

class _DesktopDropdownState extends State<DesktopDropdown> {
  final LayerLink _layerLink = LayerLink();
  OverlayEntry? _overlayEntry;

  void _show() {
    final renderBox = context.findRenderObject() as RenderBox;
    final size = renderBox.size;
    final c = AppColors.of(context);

    _overlayEntry = OverlayEntry(
      builder: (ctx) => _DropdownOverlay(
        link: _layerLink,
        triggerWidth: size.width,
        items: widget.items,
        selectedValue: widget.value,
        isDark: c.isDark,
        onSelect: (item) {
          widget.onChanged(item);
          _dismiss();
        },
        onDismiss: _dismiss,
      ),
    );
    Overlay.of(context).insert(_overlayEntry!);
  }

  void _dismiss() {
    _overlayEntry?.remove();
    _overlayEntry = null;
  }

  @override
  void dispose() {
    _dismiss();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return CompositedTransformTarget(
      link: _layerLink,
      child: GestureDetector(
        onTap: _show,
        child: widget.child,
      ),
    );
  }
}

/// Overlay 弹层：毛玻璃背景 + 选项列表
class _DropdownOverlay extends StatelessWidget {
  final LayerLink link;
  final double triggerWidth;
  final List<String> items;
  final String selectedValue;
  final bool isDark;
  final ValueChanged<String> onSelect;
  final VoidCallback onDismiss;

  const _DropdownOverlay({
    required this.link,
    required this.triggerWidth,
    required this.items,
    required this.selectedValue,
    required this.isDark,
    required this.onSelect,
    required this.onDismiss,
  });

  @override
  Widget build(BuildContext context) {
    final menuWidth = triggerWidth.clamp(140.0, 260.0);
    final bgColor = isDark
        ? const Color(0xFF2C2C2E).withValues(alpha: 0.95)
        : const Color(0xFFFFFFFF).withValues(alpha: 0.98);
    final borderColor = isDark
        ? const Color(0xFF3A3A3C)
        : const Color(0xFFD1D1D6);

    return Stack(
      children: [
        Positioned.fill(
          child: GestureDetector(
            onTap: onDismiss,
            behavior: HitTestBehavior.opaque,
            child: const SizedBox.expand(),
          ),
        ),
        CompositedTransformFollower(
          link: link,
          targetAnchor: Alignment.bottomLeft,
          followerAnchor: Alignment.topLeft,
          offset: const Offset(0, 4),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: BackdropFilter(
              filter: ImageFilter.blur(sigmaX: 20, sigmaY: 20),
              child: Container(
                width: menuWidth,
                constraints: const BoxConstraints(maxHeight: 260),
                decoration: BoxDecoration(
                  color: bgColor,
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: borderColor, width: 0.5),
                  boxShadow: [
                    BoxShadow(
                      color: const Color(0xFF000000).withValues(alpha: 0.2),
                      blurRadius: 16,
                      offset: const Offset(0, 4),
                    ),
                  ],
                ),
                child: ListView.builder(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  shrinkWrap: true,
                  itemCount: items.length,
                  itemBuilder: (_, i) => _DropdownItem(
                    label: items[i],
                    selected: items[i] == selectedValue,
                    isDark: isDark,
                    onTap: () => onSelect(items[i]),
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

class _DropdownItem extends StatefulWidget {
  final String label;
  final bool selected;
  final bool isDark;
  final VoidCallback onTap;

  const _DropdownItem({
    required this.label,
    required this.selected,
    required this.isDark,
    required this.onTap,
  });

  @override
  State<_DropdownItem> createState() => _DropdownItemState();
}

class _DropdownItemState extends State<_DropdownItem> {
  bool _hovering = false;

  @override
  Widget build(BuildContext context) {
    final hoverBg = widget.isDark
        ? const Color(0xFF007AFF).withValues(alpha: 0.2)
        : const Color(0xFF007AFF).withValues(alpha: 0.1);
    final textColor = widget.selected
        ? const Color(0xFF007AFF)
        : widget.isDark
            ? const Color(0xFFE5E5EA)
            : const Color(0xFF1C1C1E);

    return MouseRegion(
      onEnter: (_) => setState(() => _hovering = true),
      onExit: (_) => setState(() => _hovering = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
          margin: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
          decoration: BoxDecoration(
            color: _hovering ? hoverBg : CupertinoColors.transparent,
            borderRadius: BorderRadius.circular(5),
          ),
          child: Row(
            children: [
              if (widget.selected)
                const Padding(
                  padding: EdgeInsets.only(right: 6),
                  child: Icon(CupertinoIcons.checkmark,
                      size: 11, color: Color(0xFF007AFF)),
                ),
              Expanded(
                child: Text(
                  widget.label,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight:
                        widget.selected ? FontWeight.w600 : FontWeight.w400,
                    color: textColor,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
