import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../../core/glass_card.dart';
import '../../dashboard/views/dashboard_page.dart';
import '../../download/views/download_page.dart';
import '../../engine/views/engine_page.dart';
import '../../glossary/views/glossary_page.dart';
import '../../task/view_models/task_list_view_model.dart';
import '../../task/views/task_board_page.dart';
import '../../task/views/task_list_page.dart';
import '../../settings/views/settings_page.dart';
import '../../translation/views/translation_page.dart';
import '../../tts/views/tts_page.dart';

/// Liquid Glass 风格全局布局壳
class HomePage extends HookConsumerWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedIndex = useState(0);
    final showSettings = useState(false);
    final backend = ref.watch(backendProvider);
    useListenable(backend);

    // 侧栏"任务"徽章：监听任务列表，WebSocket 推送时刷新计数
    final taskCount = ref.watch(
      taskListProvider.select(
        (s) => s.tasks.where((task) => task.status < 3).length,
      ),
    );
    final ws = ref.read(wsServiceProvider);
    useEffect(() {
      Future.microtask(() => ref.read(taskListProvider.notifier).refresh());
      final sub = ws.events.listen((event) {
        if (event.type.startsWith('task_')) {
          ref.read(taskListProvider.notifier).refresh();
        }
      });
      return sub.cancel;
    }, const []);

    final themeMode = ref.watch(themeProvider);
    final brightness = MacosTheme.brightnessOf(context);
    final isDark = brightness == Brightness.dark;

    return Container(
      color: isDark ? const Color(0xFF050508) : const Color(0xFFDEE2EC),
      child: Stack(
        children: [
          // 环境光斑：供侧栏/顶栏/状态栏的玻璃材质透出色彩
          Positioned(
            left: -140,
            top: -140,
            child: _AmbientBlob(
              size: 420,
              color: isDark ? const Color(0xFF3B3BF0) : const Color(0xFFA5B8FF),
            ),
          ),
          Positioned(
            right: -100,
            top: 80,
            child: _AmbientBlob(
              size: 360,
              color: isDark ? const Color(0xFF0A7D8C) : const Color(0xFF9FE3E8),
            ),
          ),
          Positioned(
            left: 220,
            bottom: -160,
            child: _AmbientBlob(
              size: 380,
              color: isDark ? const Color(0xFF8C2F6E) : const Color(0xFFF3C2DD),
            ),
          ),
          // 主结构
          Column(
            children: [
              Expanded(
                child: Row(
                  children: [
                    _NavRail(
                      selectedIndex: selectedIndex.value,
                      showSettings: showSettings.value,
                      taskCount: taskCount,
                      engineOnline: backend.online,
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
                            title: null,
                            themeMode: themeMode,
                            onToggleTheme: () =>
                                ref.read(themeProvider.notifier).toggle(),
                          ),
                          Expanded(
                            child: ClipRect(
                              child: Navigator(
                                key: ValueKey(
                                  'content_nav_${selectedIndex.value}_${showSettings.value}',
                                ),
                                onGenerateRoute: (_) => PageRouteBuilder(
                                  pageBuilder: (_, _, _) => _buildContent(
                                    selectedIndex.value,
                                    showSettings.value,
                                  ),
                                  transitionDuration: Duration.zero,
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
            ],
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
        return const TaskBoardPage();
      case 2:
        return const DownloadPage();
      case 3:
        return const TaskListPage();
      case 4:
        return const _TaskStylePage(
          title: '校对',
          desc: '字幕编辑器：时间轴 + 原文 + 译文，播放器联动',
          icon: CupertinoIcons.pencil_ellipsis_rectangle,
        );
      case 5:
        return const _TaskStylePage(
          title: '合成',
          desc: '将字幕烧录到视频，或封装为独立字幕文件',
          icon: CupertinoIcons.film,
        );
      case 6:
        return const _TaskStylePage(
          title: '配音',
          desc: 'TTS 配音 + 声音克隆，输出完整配音视频',
          icon: CupertinoIcons.mic,
        );
      case 7:
        return const EnginePage();
      case 8:
        return const TranslationPage();
      case 9:
        return const GlossaryPage();
      case 10:
        return const TTSPage();
      default:
        return const SizedBox.shrink();
    }
  }
}

/// 环境光斑：径向渐隐圆，模拟 macOS 26 桌面透过玻璃的环境色
class _AmbientBlob extends StatelessWidget {
  final double size;
  final Color color;
  const _AmbientBlob({required this.size, required this.color});

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          gradient: RadialGradient(
            colors: [color.withValues(alpha: 0.55), color.withValues(alpha: 0)],
          ),
        ),
      ),
    );
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
  _NavSection(CupertinoIcons.square_list, '任务'),
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

/// 左侧玻璃侧栏：分组标签 + 强调色胶囊选中态
class _NavRail extends StatelessWidget {
  final int selectedIndex;
  final bool showSettings;
  final int taskCount;
  final bool engineOnline;
  final ValueChanged<int> onSelect;
  final VoidCallback onSettings;

  const _NavRail({
    required this.selectedIndex,
    required this.showSettings,
    required this.taskCount,
    required this.engineOnline,
    required this.onSelect,
    required this.onSettings,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return GlassBar(
      child: SizedBox(
        width: 180,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(height: 44),
            _GroupLabel('任务', c),
            ..._taskItems.asMap().entries.map(
              (e) => _RailItem(
                icon: e.value.icon,
                label: e.value.label,
                selected: !showSettings && selectedIndex == e.key,
                // "任务"项右侧显示任务数徽章
                badge: e.value.label == '任务' ? taskCount : null,
                onTap: () => onSelect(e.key),
              ),
            ),
            _GroupLabel('配置', c),
            ..._configItems.asMap().entries.map(
              (e) => _RailItem(
                icon: e.value.icon,
                label: e.value.label,
                selected:
                    !showSettings && selectedIndex == e.key + _taskItems.length,
                // "引擎"项右侧显示引擎在线状态点
                online: e.value.label == '引擎' ? engineOnline : null,
                onTap: () => onSelect(e.key + _taskItems.length),
              ),
            ),
            const Spacer(),
            Container(
              margin: const EdgeInsets.symmetric(horizontal: 12),
              height: 0.5,
              color: c.glassStroke,
            ),
            const SizedBox(height: 4),
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

/// 侧栏分组小标签
class _GroupLabel extends StatelessWidget {
  final String text;
  final AppColors c;
  const _GroupLabel(this.text, this.c);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 10, 10, 4),
      child: Text(
        text,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: c.textTertiary,
        ),
      ),
    );
  }
}

/// 侧栏单个导航项：选中为强调色实心胶囊，可带数量徽章与状态点
class _RailItem extends HookWidget {
  final IconData icon;
  final String label;
  final bool selected;
  final int? badge;
  final bool? online;
  final VoidCallback onTap;

  const _RailItem({
    required this.icon,
    required this.label,
    required this.selected,
    this.badge,
    this.online,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final hovering = useState(false);
    final pressing = useState(false);

    final fg = selected
        ? const Color(0xFFFFFFFF)
        : hovering.value
        ? c.textPrimary
        : c.textSecondary;

    return MouseRegion(
      cursor: SystemMouseCursors.click,
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
          height: 30,
          margin: const EdgeInsets.symmetric(horizontal: 10, vertical: 1.5),
          padding: const EdgeInsets.symmetric(horizontal: 10),
          transform: pressing.value
              ? (Matrix4.identity()..scaleByDouble(0.97, 0.97, 1.0, 1.0))
              : Matrix4.identity(),
          transformAlignment: Alignment.center,
          decoration: BoxDecoration(
            color: selected
                ? c.accent
                : hovering.value
                ? c.glassCardBg
                : const Color(0x00000000),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Icon(icon, size: 15, color: fg),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(
                    fontSize: 12.5,
                    fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                    color: fg,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              if (badge != null)
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 6,
                    vertical: 1,
                  ),
                  decoration: BoxDecoration(
                    color: selected ? const Color(0x40FFFFFF) : c.glassCardBg,
                    borderRadius: BorderRadius.circular(9),
                  ),
                  child: Text(
                    '$badge',
                    style: TextStyle(
                      fontSize: 10.5,
                      fontWeight: FontWeight.w600,
                      color: fg,
                    ),
                  ),
                ),
              if (online != null) ...[
                const SizedBox(width: 4),
                Container(
                  width: 7,
                  height: 7,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: online! ? AppColors.green : AppColors.red,
                    boxShadow: [
                      BoxShadow(
                        color: (online! ? AppColors.green : AppColors.red)
                            .withValues(alpha: 0.4),
                        blurRadius: 4,
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

/// 顶栏：含 24px 窗口拖拽区，玻璃材质无分割线
class _TopBar extends HookWidget {
  final String? title;
  final AppThemeMode themeMode;
  final VoidCallback onToggleTheme;

  const _TopBar({
    required this.title,
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
    final c = AppColors.of(context);
    final hovering = useState(false);

    return GlassBar(
      padding: const EdgeInsets.fromLTRB(22, 30, 14, 10),
      child: Row(
        children: [
          if (title != null)
            Text(
              title!,
              style: TextStyle(
                fontSize: 19,
                fontWeight: FontWeight.w700,
                color: c.textPrimary,
                letterSpacing: -0.3,
              ),
            ),
          const Spacer(),
          MouseRegion(
            cursor: SystemMouseCursors.click,
            onEnter: (_) => hovering.value = true,
            onExit: (_) => hovering.value = false,
            child: GestureDetector(
              onTap: onToggleTheme,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 120),
                padding: const EdgeInsets.all(6),
                decoration: BoxDecoration(
                  color: hovering.value
                      ? c.glassCardBg
                      : const Color(0x00000000),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(_themeIcon, size: 15, color: c.textSecondary),
                    const SizedBox(width: 4),
                    Text(
                      _themeTooltip,
                      style: TextStyle(fontSize: 11, color: c.textSecondary),
                    ),
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

/// 任务型页面（带文件导入区）
class _TaskStylePage extends StatelessWidget {
  final String title;
  final String desc;
  final IconData icon;
  const _TaskStylePage({
    required this.title,
    required this.desc,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w700,
              color: c.textPrimary,
            ),
          ),
          const SizedBox(height: 4),
          Text(desc, style: TextStyle(fontSize: 13, color: c.textSecondary)),
          const SizedBox(height: 24),
          Expanded(
            child: GlassCard(
              radius: 16,
              child: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Container(
                      width: 48,
                      height: 48,
                      decoration: BoxDecoration(
                        color: c.accent.withValues(alpha: 0.14),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Icon(icon, size: 24, color: c.accent),
                    ),
                    const SizedBox(height: 16),
                    Text(
                      '导入文件',
                      style: TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w600,
                        color: c.textPrimary,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      '支持 MP4 / MKV / MOV / MP3 / WAV 等格式',
                      style: TextStyle(fontSize: 12, color: c.textSecondary),
                    ),
                    const SizedBox(height: 16),
                    AppButton(
                      onPressed: () {},
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(CupertinoIcons.doc_fill, size: 13),
                          SizedBox(width: 6),
                          Text(
                            '导入',
                            style: TextStyle(fontWeight: FontWeight.w600),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      '也可以把媒体文件或文件夹直接拖进本页',
                      style: TextStyle(fontSize: 11, color: c.textTertiary),
                    ),
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
