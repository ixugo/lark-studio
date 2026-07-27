import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../providers.dart';
import '../../dashboard/views/dashboard_page.dart';
import '../../glossary/views/glossary_page.dart';
import '../../task/views/task_list_page.dart';
import '../../settings/views/settings_page.dart';

class HomePage extends HookConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedIndex = useState(0);
    final showSettings = useState(false);

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
            selectedIndex: selectedIndex.value,
            showSettings: showSettings.value,
            onSelect: (i) {
              selectedIndex.value = i;
              showSettings.value = false;
            },
            onSettings: () => showSettings.value = true,
          ),
          Expanded(
            child: CupertinoPageScaffold(
              backgroundColor: CupertinoColors.transparent,
              child: _buildContent(
                  selectedIndex.value, showSettings.value),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildContent(int selectedIndex, bool showSettings) {
    if (showSettings) return const SettingsPage();
    switch (selectedIndex) {
      case 0:
        return const DashboardPage();
      case 1:
        return const TaskListPage();
      case 5:
        return const GlossaryPage();
      default:
        return Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(_menuItems[selectedIndex].icon,
                  size: 48, color: const Color(0xFFC7C7CC)),
              const SizedBox(height: 12),
              Text(_menuItems[selectedIndex].label,
                  style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w600,
                      color: Color(0xFF8E8E93))),
              const SizedBox(height: 4),
              const Text('即将推出',
                  style: TextStyle(fontSize: 13, color: Color(0xFFC7C7CC))),
            ],
          ),
        );
    }
  }
}

class _MenuItem {
  final IconData icon;
  final IconData activeIcon;
  final String label;
  const _MenuItem(this.icon, this.activeIcon, this.label);
}

const _menuItems = <_MenuItem>[
  _MenuItem(CupertinoIcons.house, CupertinoIcons.house_fill, '主页'),
  _MenuItem(CupertinoIcons.arrow_down_circle,
      CupertinoIcons.arrow_down_circle_fill, '下载'),
  _MenuItem(CupertinoIcons.text_badge_checkmark,
      CupertinoIcons.text_badge_checkmark, '字幕'),
  _MenuItem(CupertinoIcons.globe, CupertinoIcons.globe, '翻译'),
  _MenuItem(CupertinoIcons.mic, CupertinoIcons.mic_fill, '配音'),
  _MenuItem(CupertinoIcons.book, CupertinoIcons.book_fill, '词库'),
];

class _Sidebar extends HookConsumerWidget {
  final int selectedIndex;
  final bool showSettings;
  final ValueChanged<int> onSelect;
  final VoidCallback onSettings;

  const _Sidebar({
    required this.selectedIndex,
    required this.showSettings,
    required this.onSelect,
    required this.onSettings,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final backend = ref.watch(backendProvider);
    useListenable(backend);

    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 40, sigmaY: 40),
        child: Container(
          width: 200,
          decoration: BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: [
                const Color(0xFFFFFFFF).withValues(alpha: 0.72),
                const Color(0xFFF2F2F7).withValues(alpha: 0.68),
              ],
            ),
            border: const Border(
                right: BorderSide(color: Color(0x22000000), width: 0.5)),
          ),
          child: SafeArea(
            child: Column(
              children: [
                const SizedBox(height: 14),
                _EngineIndicator(online: backend.online),
                const SizedBox(height: 18),
                ..._menuItems.asMap().entries.map((e) {
                  final idx = e.key;
                  final item = e.value;
                  return _NavItem(
                    icon: item.icon,
                    activeIcon: item.activeIcon,
                    label: item.label,
                    selected: !showSettings && selectedIndex == idx,
                    onTap: () => onSelect(idx),
                  );
                }),
                const Spacer(),
                _NavItem(
                  icon: CupertinoIcons.gear,
                  activeIcon: CupertinoIcons.gear_alt_fill,
                  label: '设置',
                  selected: showSettings,
                  onTap: onSettings,
                ),
                const SizedBox(height: 14),
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
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Container(
            width: 9,
            height: 9,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color:
                  online ? const Color(0xFF34C759) : const Color(0xFFFF3B30),
              boxShadow: [
                BoxShadow(
                  color: (online
                          ? const Color(0xFF34C759)
                          : const Color(0xFFFF3B30))
                      .withValues(alpha: 0.45),
                  blurRadius: 8,
                  spreadRadius: 1,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            online ? '引擎就绪' : '引擎异常',
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w500,
              color:
                  online ? const Color(0xFF8E8E93) : const Color(0xFFFF3B30),
              letterSpacing: 0.2,
            ),
          ),
        ],
      ),
    );
  }
}

class _NavItem extends HookWidget {
  final IconData icon;
  final IconData activeIcon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _NavItem({
    required this.icon,
    required this.activeIcon,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final active = selected;

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          margin: const EdgeInsets.symmetric(horizontal: 10, vertical: 1),
          padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
          decoration: BoxDecoration(
            color: active
                ? const Color(0xFF007AFF).withValues(alpha: 0.12)
                : hovering.value
                    ? const Color(0xFF000000).withValues(alpha: 0.04)
                    : CupertinoColors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Icon(
                active ? activeIcon : icon,
                size: 18,
                color: active
                    ? const Color(0xFF007AFF)
                    : const Color(0xFF8E8E93),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                    color: active
                        ? const Color(0xFF007AFF)
                        : const Color(0xFF3A3A3C),
                    letterSpacing: -0.1,
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
