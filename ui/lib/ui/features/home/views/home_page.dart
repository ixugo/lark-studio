import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../dashboard/views/dashboard_page.dart';
import '../../download/views/download_page.dart';
import '../../engine/views/engine_page.dart';
import '../../glossary/views/glossary_page.dart';
import '../../task/views/task_list_page.dart';
import '../../settings/views/settings_page.dart';
import '../../translation/views/translation_page.dart';
import '../../tts/views/tts_page.dart';

/// SmartSub 风格全局布局壳
class HomePage extends HookConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedIndex = useState(0);
    final showSettings = useState(false);
    final backend = ref.watch(backendProvider);
    useListenable(backend);

    final themeMode = ref.watch(themeProvider);
    final brightness = MacosTheme.brightnessOf(context);
    final isDark = brightness == Brightness.dark;

    final bgColor = isDark ? const Color(0xFF000000) : const Color(0xFFF2F2F7);
    final contentBg =
        isDark ? const Color(0xFF0A0A0A) : const Color(0xFFFFFFFF);

    return Container(
      color: bgColor,
      child: Column(
        children: [
          Expanded(
            child: Row(
              children: [
                _NavRail(
                  selectedIndex: selectedIndex.value,
                  showSettings: showSettings.value,
                  isDark: isDark,
                  onSelect: (i) {
                    selectedIndex.value = i;
                    showSettings.value = false;
                  },
                  onSettings: () => showSettings.value = true,
                ),
                Expanded(
                  child: Column(
                    children: [
                      SizedBox(
                          height: 28,
                          child: Container(color: isDark
                              ? const Color(0xFF0D0D0D)
                              : const Color(0xFFFFFFFF))),
                      _TopBar(
                        title: showSettings.value
                            ? '设置'
                            : _navSections[selectedIndex.value].label,
                        isDark: isDark,
                        themeMode: themeMode,
                        onToggleTheme: () =>
                            ref.read(themeProvider.notifier).toggle(),
                      ),
                      Expanded(
                        child: Container(
                          decoration: BoxDecoration(color: contentBg),
                          child: ClipRect(
                            child: Navigator(
                              key: ValueKey('content_nav_${selectedIndex.value}_${showSettings.value}'),
                              onGenerateRoute: (_) => PageRouteBuilder(
                                pageBuilder: (_, __, ___) => _buildContent(
                                    selectedIndex.value, showSettings.value),
                                transitionDuration: Duration.zero,
                              ),
                            ),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          _StatusBar(online: backend.online, isDark: isDark),
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
        return const DownloadPage();
      case 2:
        return const TaskListPage();
      case 3:
        return const _TaskStylePage(
          title: '校对',
          desc: '字幕编辑器：时间轴 + 原文 + 译文，播放器联动',
          icon: CupertinoIcons.pencil_ellipsis_rectangle,
        );
      case 4:
        return const _TaskStylePage(
          title: '合成',
          desc: '将字幕烧录到视频，或封装为独立字幕文件',
          icon: CupertinoIcons.film,
        );
      case 5:
        return const _TaskStylePage(
          title: '配音',
          desc: 'TTS 配音 + 声音克隆，输出完整配音视频',
          icon: CupertinoIcons.mic,
        );
      case 6:
        return const EnginePage();
      case 7:
        return const TranslationPage();
      case 8:
        return const GlossaryPage();
      case 9:
        return const TTSPage();
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

/// 左侧 160px 横排导航 sidebar
class _NavRail extends HookWidget {
  final int selectedIndex;
  final bool showSettings;
  final bool isDark;
  final ValueChanged<int> onSelect;
  final VoidCallback onSettings;

  const _NavRail({
    required this.selectedIndex,
    required this.showSettings,
    required this.isDark,
    required this.onSelect,
    required this.onSettings,
  });

  @override
  Widget build(BuildContext context) {
    final railBg = isDark ? const Color(0xFF0D0D0D) : const Color(0xFFFFFFFF);
    final borderColor =
        isDark ? const Color(0xFF1F1F1F) : const Color(0xFFE5E5EA);

    return Container(
      width: 160,
      decoration: BoxDecoration(
        color: railBg,
        border: Border(right: BorderSide(color: borderColor, width: 0.5)),
      ),
      child: Column(
          children: [
            const SizedBox(height: 38),
            // 任务组
            ..._taskItems.asMap().entries.map((e) => _RailItem(
                  icon: e.value.icon,
                  label: e.value.label,
                  selected: !showSettings && selectedIndex == e.key,
                  isDark: isDark,
                  onTap: () => onSelect(e.key),
                )),
            // 分隔线
            Container(
              margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              height: 0.5,
              color: isDark ? const Color(0xFF2C2C2E) : const Color(0xFFE5E5EA),
            ),
            // 配置组
            ..._configItems.asMap().entries.map((e) => _RailItem(
                  icon: e.value.icon,
                  label: e.value.label,
                  selected:
                      !showSettings && selectedIndex == e.key + _taskItems.length,
                  isDark: isDark,
                  onTap: () => onSelect(e.key + _taskItems.length),
                )),
            const Spacer(),
            // 分隔线
            Container(
              margin: const EdgeInsets.symmetric(horizontal: 12),
              height: 0.5,
              color: isDark ? const Color(0xFF2C2C2E) : const Color(0xFFE5E5EA),
            ),
            const SizedBox(height: 4),
            // 设置
            _RailItem(
              icon: CupertinoIcons.gear,
              label: '设置',
              selected: showSettings,
              isDark: isDark,
              onTap: onSettings,
            ),
            const SizedBox(height: 8),
          ],
      ),
    );
  }
}

/// Sidebar 单个导航项：左图右字，选中态左缘指示条
class _RailItem extends HookWidget {
  final IconData icon;
  final String label;
  final bool selected;
  final bool isDark;
  final VoidCallback onTap;

  const _RailItem({
    required this.icon,
    required this.label,
    required this.selected,
    this.isDark = true,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final pressing = useState(false);

    final hoverBg = isDark
        ? const Color(0xFFFFFFFF).withValues(alpha: 0.05)
        : const Color(0xFF000000).withValues(alpha: 0.04);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTapDown: (_) => pressing.value = true,
        onTapUp: (_) => pressing.value = false,
        onTapCancel: () => pressing.value = false,
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          curve: Curves.easeOut,
          height: 32,
          margin: const EdgeInsets.symmetric(horizontal: 8, vertical: 1),
          padding: const EdgeInsets.symmetric(horizontal: 10),
          transform: pressing.value
              ? (Matrix4.identity()..scaleByDouble(0.97, 0.97, 1.0, 1.0))
              : Matrix4.identity(),
          transformAlignment: Alignment.center,
          decoration: BoxDecoration(
            color: selected
                ? const Color(0xFF007AFF).withValues(alpha: 0.12)
                : hovering.value
                    ? hoverBg
                    : const Color(0x00000000),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Row(
            children: [
              if (selected)
                Container(
                  width: 3,
                  height: 14,
                  margin: const EdgeInsets.only(right: 8),
                  decoration: BoxDecoration(
                    color: const Color(0xFF007AFF),
                    borderRadius: BorderRadius.circular(1.5),
                  ),
                )
              else
                const SizedBox(width: 11),
              Icon(
                icon,
                size: 15,
                color: selected
                    ? const Color(0xFF007AFF)
                    : hovering.value
                        ? (isDark ? const Color(0xFFE5E5EA) : const Color(0xFF1C1C1E))
                        : const Color(0xFF8E8E93),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                    color: selected
                        ? const Color(0xFF007AFF)
                        : hovering.value
                            ? (isDark ? const Color(0xFFE5E5EA) : const Color(0xFF1C1C1E))
                            : const Color(0xFF8E8E93),
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 顶栏 (44px)：页面名 + 主题切换
class _TopBar extends StatelessWidget {
  final String title;
  final bool isDark;
  final AppThemeMode themeMode;
  final VoidCallback onToggleTheme;

  const _TopBar({
    required this.title,
    required this.isDark,
    required this.themeMode,
    required this.onToggleTheme,
  });

  IconData get _themeIcon => switch (themeMode) {
        AppThemeMode.dark => CupertinoIcons.moon_fill,
        AppThemeMode.light => CupertinoIcons.sun_max_fill,
        AppThemeMode.system => CupertinoIcons.circle_lefthalf_fill,
      };

  String get _themeTooltip => switch (themeMode) {
        AppThemeMode.dark => '暗色',
        AppThemeMode.light => '亮色',
        AppThemeMode.system => '跟随系统',
      };

  @override
  Widget build(BuildContext context) {
    final barBg = isDark ? const Color(0xFF0D0D0D) : const Color(0xFFFFFFFF);
    final borderColor =
        isDark ? const Color(0xFF1F1F1F) : const Color(0xFFE5E5EA);
    final titleColor =
        isDark ? const Color(0xFF8E8E93) : const Color(0xFF636366);

    return Container(
      height: 44,
      decoration: BoxDecoration(
        color: barBg,
        border: Border(bottom: BorderSide(color: borderColor, width: 0.5)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        children: [
          Text(title,
              style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: titleColor)),
          const Spacer(),
          GestureDetector(
            onTap: onToggleTheme,
            child: MouseRegion(
              cursor: SystemMouseCursors.click,
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(_themeIcon, size: 15, color: titleColor),
                  const SizedBox(width: 4),
                  Text(_themeTooltip,
                      style: TextStyle(fontSize: 11, color: titleColor)),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// 底部状态栏 (26px)
class _StatusBar extends StatelessWidget {
  final bool online;
  final bool isDark;
  const _StatusBar({required this.online, required this.isDark});

  @override
  Widget build(BuildContext context) {
    final barBg = isDark ? const Color(0xFF0D0D0D) : const Color(0xFFFFFFFF);
    final borderColor =
        isDark ? const Color(0xFF1F1F1F) : const Color(0xFFE5E5EA);
    final secondaryText =
        isDark ? const Color(0xFF8E8E93) : const Color(0xFF636366);

    return Container(
      height: 26,
      decoration: BoxDecoration(
        color: barBg,
        border: Border(top: BorderSide(color: borderColor, width: 0.5)),
      ),
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Row(
        children: [
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
            online ? '引擎就绪' : '引擎离线',
            style: TextStyle(
              fontSize: 11,
              color: online ? secondaryText : const Color(0xFFFF3B30),
            ),
          ),
          const SizedBox(width: 16),
          Text('GPU: CoreML',
              style: TextStyle(fontSize: 11, color: secondaryText)),
          const Spacer(),
          Text('v0.1.0',
              style: TextStyle(
                  fontSize: 10,
                  fontFamily: 'monospace',
                  color: isDark
                      ? const Color(0xFF48484A)
                      : const Color(0xFFC7C7CC))),
        ],
      ),
    );
  }
}

/// 任务型页面（带文件导入区）
class _TaskStylePage extends StatelessWidget {
  final String title;
  final String desc;
  final IconData icon;
  const _TaskStylePage(
      {required this.title, required this.desc, required this.icon});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title,
              style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w700,
                  color: c.textPrimary)),
          const SizedBox(height: 4),
          Text(desc,
              style: TextStyle(fontSize: 13, color: c.textSecondary)),
          const SizedBox(height: 24),
          Expanded(
            child: Container(
              decoration: BoxDecoration(
                color: c.cardBg,
                borderRadius: BorderRadius.circular(14),
                border: Border.all(
                  color: AppColors.blue.withValues(alpha: 0.3),
                  width: 1,
                  strokeAlign: BorderSide.strokeAlignInside,
                ),
              ),
              child: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      width: 48,
                      height: 48,
                      decoration: BoxDecoration(
                        color: AppColors.blue.withValues(alpha: 0.1),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(icon, size: 24,
                          color: AppColors.blue.withValues(alpha: 0.7)),
                    ),
                    const SizedBox(height: 16),
                    Text('导入文件',
                        style: TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                            color: c.textPrimary)),
                    const SizedBox(height: 6),
                    Text('支持 MP4 / MKV / MOV / MP3 / WAV 等格式',
                        style: TextStyle(fontSize: 12, color: c.textSecondary)),
                    const SizedBox(height: 16),
                    AppButton(
                      onPressed: () {},
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(CupertinoIcons.doc_fill, size: 13),
                          SizedBox(width: 6),
                          Text('导入', style: TextStyle(fontWeight: FontWeight.w600)),
                        ],
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text('也可以把媒体文件或文件夹直接拖进本页',
                        style: TextStyle(fontSize: 11, color: c.textTertiary)),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
