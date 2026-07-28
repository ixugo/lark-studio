import 'dart:ui';
import 'package:flutter/cupertino.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../providers.dart';
import '../../dashboard/views/dashboard_page.dart';
import '../../glossary/views/glossary_page.dart';
import '../../task/views/task_list_page.dart';
import '../../settings/views/settings_page.dart';

/// SmartSub 风格全局布局壳
class HomePage extends HookConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedIndex = useState(0);
    final showSettings = useState(false);
    final backend = ref.watch(backendProvider);
    useListenable(backend);

    return Container(
      color: const Color(0xFF000000),
      child: Column(
        children: [
          Expanded(
            child: Row(
              children: [
                _NavRail(
                  selectedIndex: selectedIndex.value,
                  showSettings: showSettings.value,
                  onSelect: (i) {
                    selectedIndex.value = i;
                    showSettings.value = false;
                  },
                  onSettings: () => showSettings.value = true,
                ),
                Expanded(
                  child: Column(
                    children: [
                      _TopBar(
                        title: showSettings.value
                            ? '设置'
                            : _navSections[selectedIndex.value].label,
                      ),
                      Expanded(
                        child: Container(
                          decoration: const BoxDecoration(
                            color: Color(0xFF0A0A0A),
                          ),
                          child: _buildContent(
                              selectedIndex.value, showSettings.value),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          _StatusBar(online: backend.online),
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
        return const _PlaceholderPage(icon: CupertinoIcons.arrow_down_circle, label: '下载');
      case 2:
        return const TaskListPage();
      case 3:
        return const _PlaceholderPage(icon: CupertinoIcons.pencil_ellipsis_rectangle, label: '校对');
      case 4:
        return const _PlaceholderPage(icon: CupertinoIcons.film, label: '合成');
      case 5:
        return const _PlaceholderPage(icon: CupertinoIcons.mic, label: '配音');
      case 6:
        return const _PlaceholderPage(icon: CupertinoIcons.bolt, label: '引擎');
      case 7:
        return const _PlaceholderPage(icon: CupertinoIcons.globe, label: '翻译');
      case 8:
        return const GlossaryPage();
      case 9:
        return const _PlaceholderPage(icon: CupertinoIcons.waveform, label: '音色');
      default:
        return const SizedBox.shrink();
    }
  }
}

/// 导航分组定义
class _NavSection {
  final IconData icon;
  final String label;
  const _NavSection(this.icon, this.label);
}

/// 任务组 + 配置组导航项
const _taskItems = <_NavSection>[
  _NavSection(CupertinoIcons.house, '启动台'),
  _NavSection(CupertinoIcons.arrow_down_circle, '下载'),
  _NavSection(CupertinoIcons.captions_bubble, '字幕'),
  _NavSection(CupertinoIcons.pencil_ellipsis_rectangle, '校对'),
  _NavSection(CupertinoIcons.film, '合成'),
  _NavSection(CupertinoIcons.mic, '配音'),
];

const _configItems = <_NavSection>[
  _NavSection(CupertinoIcons.bolt, '引擎'),
  _NavSection(CupertinoIcons.globe, '翻译'),
  _NavSection(CupertinoIcons.book, '词库'),
  _NavSection(CupertinoIcons.waveform, '音色'),
];

const _navSections = [..._taskItems, ..._configItems];

/// 左侧 64px 竖排导航 rail
class _NavRail extends HookWidget {
  final int selectedIndex;
  final bool showSettings;
  final ValueChanged<int> onSelect;
  final VoidCallback onSettings;

  const _NavRail({
    required this.selectedIndex,
    required this.showSettings,
    required this.onSelect,
    required this.onSettings,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 64,
      decoration: const BoxDecoration(
        color: Color(0xFF0D0D0D),
        border: Border(
            right: BorderSide(color: Color(0xFF1F1F1F), width: 0.5)),
      ),
      child: SafeArea(
        child: Column(
          children: [
            const SizedBox(height: 10),
            // Logo
            Container(
              width: 32,
              height: 32,
              margin: const EdgeInsets.only(bottom: 12),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(8),
                gradient: const LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [Color(0xFF007AFF), Color(0xFF5856D6)],
                ),
              ),
              child: const Center(
                child: Text('S',
                    style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w800,
                        color: CupertinoColors.white)),
              ),
            ),
            // 任务组
            ..._taskItems.asMap().entries.map((e) => _RailItem(
                  icon: e.value.icon,
                  label: e.value.label,
                  selected: !showSettings && selectedIndex == e.key,
                  onTap: () => onSelect(e.key),
                )),
            // 分隔线
            Container(
              width: 28,
              height: 0.5,
              margin: const EdgeInsets.symmetric(vertical: 6),
              color: const Color(0xFF2C2C2E),
            ),
            // 配置组
            ..._configItems.asMap().entries.map((e) => _RailItem(
                  icon: e.value.icon,
                  label: e.value.label,
                  selected:
                      !showSettings && selectedIndex == e.key + _taskItems.length,
                  onTap: () => onSelect(e.key + _taskItems.length),
                )),
            const Spacer(),
            // 分隔线
            Container(
              width: 28,
              height: 0.5,
              margin: const EdgeInsets.only(bottom: 4),
              color: const Color(0xFF2C2C2E),
            ),
            // 设置
            _RailItem(
              icon: CupertinoIcons.gear,
              label: '设置',
              selected: showSettings,
              onTap: onSettings,
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }
}

/// Rail 单个导航项：图标在上、文字在下，选中态左缘指示条
class _RailItem extends HookWidget {
  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onTap;

  const _RailItem({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 150),
          curve: Curves.easeOut,
          width: 52,
          height: 48,
          margin: const EdgeInsets.symmetric(vertical: 1),
          decoration: BoxDecoration(
            color: selected
                ? const Color(0xFF007AFF).withValues(alpha: 0.12)
                : hovering.value
                    ? const Color(0xFFFFFFFF).withValues(alpha: 0.05)
                    : CupertinoColors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Stack(
            children: [
              // 左缘指示条
              if (selected)
                Positioned(
                  left: -4,
                  top: 12,
                  bottom: 12,
                  child: Container(
                    width: 3,
                    decoration: BoxDecoration(
                      color: const Color(0xFF007AFF),
                      borderRadius: BorderRadius.circular(1.5),
                    ),
                  ),
                ),
              Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      icon,
                      size: 19,
                      color: selected
                          ? const Color(0xFF007AFF)
                          : hovering.value
                              ? const Color(0xFFE5E5EA)
                              : const Color(0xFF8E8E93),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      label,
                      style: TextStyle(
                        fontSize: 10,
                        fontWeight:
                            selected ? FontWeight.w600 : FontWeight.w500,
                        color: selected
                            ? const Color(0xFF007AFF)
                            : hovering.value
                                ? const Color(0xFFE5E5EA)
                                : const Color(0xFF8E8E93),
                        letterSpacing: 0.1,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 顶栏 (44px)：页面名 + 搜索框占位 + 主题切换占位
class _TopBar extends StatelessWidget {
  final String title;
  const _TopBar({required this.title});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 44,
      decoration: const BoxDecoration(
        color: Color(0xFF0D0D0D),
        border:
            Border(bottom: BorderSide(color: Color(0xFF1F1F1F), width: 0.5)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Text(title,
              style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: Color(0xFF8E8E93))),
          const Spacer(),
          // 搜索框占位
          Container(
            width: 200,
            height: 28,
            decoration: BoxDecoration(
              color: const Color(0xFF1C1C1E),
              borderRadius: BorderRadius.circular(6),
              border: Border.all(color: const Color(0xFF2C2C2E), width: 0.5),
            ),
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: const Row(
              children: [
                Icon(CupertinoIcons.search, size: 13, color: Color(0xFF636366)),
                SizedBox(width: 6),
                Text('搜索或跳转...',
                    style: TextStyle(fontSize: 11, color: Color(0xFF636366))),
                Spacer(),
                Text('⌘K',
                    style: TextStyle(
                        fontSize: 10,
                        fontFamily: 'monospace',
                        color: Color(0xFF48484A))),
              ],
            ),
          ),
          const SizedBox(width: 12),
          // 主题切换按钮
          GestureDetector(
            child: const Icon(CupertinoIcons.moon,
                size: 16, color: Color(0xFF8E8E93)),
          ),
        ],
      ),
    );
  }
}

/// 底部状态栏 (26px)
class _StatusBar extends StatelessWidget {
  final bool online;
  const _StatusBar({required this.online});

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 26,
      decoration: const BoxDecoration(
        color: Color(0xFF0D0D0D),
        border: Border(top: BorderSide(color: Color(0xFF1F1F1F), width: 0.5)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Row(
        children: [
          // 引擎状态
          Container(
            width: 7,
            height: 7,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color:
                  online ? const Color(0xFF34C759) : const Color(0xFFFF3B30),
              boxShadow: [
                BoxShadow(
                  color: (online
                          ? const Color(0xFF34C759)
                          : const Color(0xFFFF3B30))
                      .withValues(alpha: 0.35),
                  blurRadius: 4,
                ),
              ],
            ),
          ),
          const SizedBox(width: 6),
          Text(
            '引擎就绪',
            style: TextStyle(
              fontSize: 11,
              color: online ? const Color(0xFF8E8E93) : const Color(0xFFFF3B30),
            ),
          ),
          const SizedBox(width: 16),
          const Text('GPU: CoreML',
              style: TextStyle(fontSize: 11, color: Color(0xFF8E8E93))),
          const Spacer(),
          const Text('v0.1.0',
              style: TextStyle(
                  fontSize: 10,
                  fontFamily: 'monospace',
                  color: Color(0xFF48484A))),
        ],
      ),
    );
  }
}

/// 占位页面
class _PlaceholderPage extends StatelessWidget {
  final IconData icon;
  final String label;
  const _PlaceholderPage({required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 48, color: const Color(0xFF48484A)),
          const SizedBox(height: 12),
          Text(label,
              style: const TextStyle(
                  fontSize: 17,
                  fontWeight: FontWeight.w600,
                  color: Color(0xFF8E8E93))),
          const SizedBox(height: 4),
          const Text('即将推出',
              style: TextStyle(fontSize: 13, color: Color(0xFF48484A))),
        ],
      ),
    );
  }
}
