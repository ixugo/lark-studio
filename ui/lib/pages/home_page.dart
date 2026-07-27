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
    return Container(
      decoration: const BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [Color(0xFFF8F8FA), Color(0xFFF0F0F5), Color(0xFFECECF4)],
        ),
      ),
      child: Row(
        children: [
          _Sidebar(
            selectedIndex: _selectedIndex,
            onSelect: (i) => setState(() => _selectedIndex = i),
          ),
          Expanded(
            child: CupertinoPageScaffold(
              backgroundColor: CupertinoColors.transparent,
              child: _pages[_selectedIndex],
            ),
          ),
        ],
      ),
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
        filter: ImageFilter.blur(sigmaX: 40, sigmaY: 40),
        child: Container(
          width: 68,
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: [
                const Color(0xFFFFFFFF).withValues(alpha: 0.72),
                const Color(0xFFF2F2F7).withValues(alpha: 0.68),
              ],
            ),
            border: const Border(right: BorderSide(color: Color(0x22000000), width: 0.5)),
          ),
          child: SafeArea(
            child: Column(
              children: [
                const SizedBox(height: 14),
                _EngineIndicator(online: backend.online),
                const SizedBox(height: 18),
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

class _EngineIndicator extends StatelessWidget {
  final bool online;
  const _EngineIndicator({required this.online});

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Container(
          width: 9, height: 9,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: online ? const Color(0xFF34C759) : const Color(0xFFFF3B30),
            boxShadow: [
              BoxShadow(
                color: (online ? const Color(0xFF34C759) : const Color(0xFFFF3B30)).withValues(alpha: 0.45),
                blurRadius: 8,
                spreadRadius: 1,
              ),
            ],
          ),
        ),
        const SizedBox(height: 3),
        Text(
          online ? '就绪' : '异常',
          style: TextStyle(
            fontSize: 9,
            fontWeight: FontWeight.w500,
            color: online ? const Color(0xFF8E8E93) : const Color(0xFFFF3B30),
            letterSpacing: 0.2,
          ),
        ),
      ],
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
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          margin: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
          padding: const EdgeInsets.symmetric(vertical: 10),
          decoration: BoxDecoration(
            color: active
                ? const Color(0xFF007AFF).withValues(alpha: 0.12)
                : _hovering
                    ? const Color(0xFF000000).withValues(alpha: 0.04)
                    : CupertinoColors.transparent,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(widget.icon, size: 21, color: active ? const Color(0xFF007AFF) : const Color(0xFF8E8E93)),
              const SizedBox(height: 3),
              Text(widget.label, style: TextStyle(
                fontSize: 10,
                fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                color: active ? const Color(0xFF007AFF) : const Color(0xFF8E8E93),
                letterSpacing: 0.1,
              )),
            ],
          ),
        ),
      ),
    );
  }
}
