import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../services/backend_service.dart';
import 'task_list_page.dart';
import 'settings_page.dart';

/// macOS 风格侧边栏主页
class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _selectedIndex = 0;

  static const _pages = <Widget>[
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
            backgroundColor: CupertinoColors.systemGroupedBackground,
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

  @override
  Widget build(BuildContext context) {
    final backend = context.watch<BackendService>();

    return Container(
      width: 220,
      decoration: const BoxDecoration(
        color: Color(0xFFF5F5F7),
        border: Border(right: BorderSide(color: Color(0xFFD1D1D6), width: 0.5)),
      ),
      child: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 16, 20, 4),
              child: Row(
                children: [
                  const Text(
                    'vdub',
                    style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700, color: Color(0xFF1D1D1F), letterSpacing: -0.5),
                  ),
                  const Spacer(),
                  Container(
                    width: 8, height: 8,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: backend.online ? CupertinoColors.systemGreen : CupertinoColors.systemRed,
                    ),
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
              child: Text(
                backend.online ? '后端已连接' : '后端未连接',
                style: TextStyle(fontSize: 11, color: backend.online ? const Color(0xFF8E8E93) : CupertinoColors.systemRed),
              ),
            ),
            _SidebarItem(
              icon: CupertinoIcons.play_rectangle,
              label: '任务',
              selected: selectedIndex == 0,
              onTap: () => onSelect(0),
            ),
            _SidebarItem(
              icon: CupertinoIcons.gear,
              label: '设置',
              selected: selectedIndex == 1,
              onTap: () => onSelect(1),
            ),
          ],
        ),
      ),
    );
  }
}

class _SidebarItem extends StatelessWidget {
  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _SidebarItem({required this.icon, required this.label, required this.selected, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? CupertinoColors.systemBlue.withValues(alpha: 0.12) : null,
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(icon, size: 18, color: selected ? CupertinoColors.systemBlue : const Color(0xFF8E8E93)),
            const SizedBox(width: 10),
            Text(label, style: TextStyle(
              fontSize: 14,
              fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
              color: selected ? CupertinoColors.systemBlue : const Color(0xFF3A3A3C),
            )),
          ],
        ),
      ),
    );
  }
}
