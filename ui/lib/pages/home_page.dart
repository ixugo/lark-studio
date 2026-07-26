import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../services/backend_service.dart';
import 'dashboard_page.dart';
import 'task_list_page.dart';
import 'settings_page.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _selectedIndex = 0;

  static const _pages = <Widget>[
    DashboardPage(),
    TaskListPage(),
    SettingsPage(),
  ];

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        _Sidebar(
          selectedIndex: _selectedIndex,
          onSelect: (i) => setState(() => _selectedIndex = i),
        ),
        Expanded(
          child: CupertinoPageScaffold(
            backgroundColor: const Color(0xFFF5F5F7),
            child: _pages[_selectedIndex],
          ),
        ),
      ],
    );
  }
}

class _Sidebar extends StatelessWidget {
  final int selectedIndex;
  final ValueChanged<int> onSelect;

  const _Sidebar({required this.selectedIndex, required this.onSelect});

  static const _items = [
    (CupertinoIcons.house_fill, '首页'),
    (CupertinoIcons.play_rectangle_fill, '任务'),
    (CupertinoIcons.gear_alt_fill, '设置'),
  ];

  @override
  Widget build(BuildContext context) {
    final backend = context.watch<BackendService>();

    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          width: 72,
          decoration: BoxDecoration(
            color: const Color(0xFFF2F2F7).withValues(alpha: 0.85),
            border: const Border(right: BorderSide(color: Color(0xFFD1D1D6), width: 0.5)),
          ),
          child: SafeArea(
            child: Column(
              children: [
                const SizedBox(height: 12),
                // 引擎状态指示灯
                Container(
                  width: 8, height: 8,
                  margin: const EdgeInsets.only(bottom: 4),
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: backend.online ? CupertinoColors.systemGreen : CupertinoColors.systemRed,
                    boxShadow: [
                      BoxShadow(
                        color: (backend.online ? CupertinoColors.systemGreen : CupertinoColors.systemRed).withValues(alpha: 0.5),
                        blurRadius: 6,
                      ),
                    ],
                  ),
                ),
                Text(
                  backend.online ? '就绪' : '异常',
                  style: TextStyle(
                    fontSize: 9,
                    fontWeight: FontWeight.w500,
                    color: backend.online ? const Color(0xFF8E8E93) : CupertinoColors.systemRed,
                  ),
                ),
                const SizedBox(height: 20),
                ..._items.asMap().entries.map((e) {
                  final idx = e.key;
                  final (icon, label) = e.value;
                  return _NavItem(
                    icon: icon,
                    label: label,
                    selected: selectedIndex == idx,
                    onTap: () => onSelect(idx),
                  );
                }),
                const Spacer(),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _NavItem extends StatefulWidget {
  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _NavItem({required this.icon, required this.label, required this.selected, required this.onTap});

  @override
  State<_NavItem> createState() => _NavItemState();
}

class _NavItemState extends State<_NavItem> {
  bool _hovering = false;

  @override
  Widget build(BuildContext context) {
    final active = widget.selected;
    return MouseRegion(
      onEnter: (_) => setState(() => _hovering = true),
      onExit: (_) => setState(() => _hovering = false),
      child: GestureDetector(
        onTap: widget.onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
          padding: const EdgeInsets.symmetric(vertical: 10),
          decoration: BoxDecoration(
            color: active
                ? CupertinoColors.systemBlue.withValues(alpha: 0.15)
                : _hovering
                    ? const Color(0xFF8E8E93).withValues(alpha: 0.08)
                    : null,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.icon, size: 22, color: active ? CupertinoColors.systemBlue : const Color(0xFF8E8E93)),
              const SizedBox(height: 3),
              Text(widget.label, style: TextStyle(
                fontSize: 10,
                fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                color: active ? CupertinoColors.systemBlue : const Color(0xFF8E8E93),
              )),
            ],
          ),
        ),
      ),
    );
  }
}
