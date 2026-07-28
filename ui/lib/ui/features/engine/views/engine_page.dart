import 'dart:ui';
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../core/app_colors.dart';
import '../../../core/app_button.dart';
import '../view_models/engine_view_model.dart';

/// 引擎配置页 — SmartSub 风格左右分栏
class EnginePage extends HookConsumerWidget {
  const EnginePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final es = ref.watch(engineProvider);
    final notifier = ref.read(engineProvider.notifier);

    useEffect(() {
      Future.microtask(() => notifier.load());
      return null;
    }, const []);

    return Row(
      children: [
        _EngineList(
          engines: es.engines,
          selectedId: es.selectedEngineId,
          onSelect: notifier.selectEngine,
        ),
        Expanded(
          child: es.selectedEngineId == null
              ? _OverviewPanel(
                  whisperMode: es.whisperMode,
                  runtime: es.whisperRuntime,
                  installing: es.installing,
                  installMessage: es.installMessage,
                  onInstall: notifier.installWhisper,
                )
              : _EngineDetail(
                  engine: es.engines.firstWhere(
                    (e) => e.id == es.selectedEngineId,
                    orElse: () => EngineItem.empty,
                  ),
                  runtime: es.whisperRuntime,
                  installing: es.installing,
                  installMessage: es.installMessage,
                  onInstall: notifier.installWhisper,
                ),
        ),
      ],
    );
  }
}

// ---- 左侧引擎列表 ----

class _EngineList extends HookWidget {
  final List<EngineItem> engines;
  final String? selectedId;
  final ValueChanged<String?> onSelect;

  const _EngineList({
    required this.engines,
    required this.selectedId,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final local = engines.where((e) => e.category == 'local').toList();
    final localMulti = engines
        .where((e) => e.category == 'local_multi')
        .toList();
    final localCmd = engines.where((e) => e.category == 'local_cmd').toList();
    final cloud = engines.where((e) => e.category == 'cloud').toList();

    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          width: 260,
          decoration: BoxDecoration(
            color: c.barBg.withValues(alpha: 0.9),
            border: Border(right: BorderSide(color: c.borderLight, width: 0.5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
                child: Row(
                  children: [
                    Text(
                      '听写引擎',
                      style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                        color: c.textPrimary,
                      ),
                    ),
                    const Spacer(),
                    Icon(CupertinoIcons.plus, size: 16, color: c.textSecondary),
                  ],
                ),
              ),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  children: [
                    _buildOverviewItem(c),
                    if (local.isNotEmpty) ...[
                      _sectionTitle('本地引擎', c),
                      ...local.map(
                        (e) => _EngineItemWidget(
                          engine: e,
                          selected: e.id == selectedId,
                          onTap: () => onSelect(e.id),
                        ),
                      ),
                    ],
                    if (localMulti.isNotEmpty) ...[
                      _sectionTitle('本地多模型引擎', c),
                      ...localMulti.map(
                        (e) => _EngineItemWidget(
                          engine: e,
                          selected: e.id == selectedId,
                          onTap: () => onSelect(e.id),
                        ),
                      ),
                    ],
                    if (localCmd.isNotEmpty) ...[
                      _sectionTitle('本地命令行', c),
                      ...localCmd.map(
                        (e) => _EngineItemWidget(
                          engine: e,
                          selected: e.id == selectedId,
                          onTap: () => onSelect(e.id),
                        ),
                      ),
                    ],
                    if (cloud.isNotEmpty) ...[
                      _sectionTitle('云端听写', c),
                      ...cloud.map(
                        (e) => _EngineItemWidget(
                          engine: e,
                          selected: e.id == selectedId,
                          onTap: () => onSelect(e.id),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildOverviewItem(AppColors c) {
    final isSelected = selectedId == null;
    return GestureDetector(
      onTap: () => onSelect(null),
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 2),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: isSelected
              ? AppColors.blue.withValues(alpha: 0.15)
              : const Color(0x00000000),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              CupertinoIcons.squares_below_rectangle,
              size: 16,
              color: c.textTertiary,
            ),
            const SizedBox(width: 10),
            Text('总览', style: TextStyle(fontSize: 13, color: c.textPrimary)),
            const SizedBox(width: 6),
            Text(
              '就绪状态与起步建议',
              style: TextStyle(fontSize: 10, color: c.textTertiary),
            ),
          ],
        ),
      ),
    );
  }

  Widget _sectionTitle(String title, AppColors c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: Text(
        title,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: c.textTertiary,
        ),
      ),
    );
  }
}

class _EngineItemWidget extends HookWidget {
  final EngineItem engine;
  final bool selected;
  final VoidCallback onTap;
  const _EngineItemWidget({
    required this.engine,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final c = AppColors.of(context);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 120),
          margin: const EdgeInsets.symmetric(vertical: 1),
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
          decoration: BoxDecoration(
            color: selected
                ? AppColors.blue.withValues(alpha: 0.15)
                : hovering.value
                ? c.cardBgHover
                : const Color(0x00000000),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Icon(engine.icon, size: 16, color: engine.color),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      engine.name,
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: selected
                            ? FontWeight.w600
                            : FontWeight.w400,
                        color: selected ? AppColors.blue : c.textPrimary,
                      ),
                    ),
                    if (engine.tags.isNotEmpty)
                      Wrap(
                        spacing: 4,
                        children: engine.tags
                            .map(
                              (t) => Text(
                                t,
                                style: TextStyle(
                                  fontSize: 9,
                                  color: c.textTertiary,
                                ),
                              ),
                            )
                            .toList(),
                      ),
                  ],
                ),
              ),
              Container(
                width: 7,
                height: 7,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: engine.available ? AppColors.green : c.borderSubtle,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---- 右侧总览面板 ----

class _OverviewPanel extends StatelessWidget {
  final String whisperMode;
  final Map<String, dynamic> runtime;
  final bool installing;
  final String installMessage;
  final VoidCallback onInstall;

  const _OverviewPanel({
    required this.whisperMode,
    required this.runtime,
    required this.installing,
    required this.installMessage,
    required this.onInstall,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final installed = runtime['installed'] as bool? ?? false;
    final acceleration = runtime['acceleration'] as String? ?? '';
    final version = runtime['version'] as String? ?? '';
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'whisper.cpp（内置）',
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.w700,
              color: c.textPrimary,
            ),
          ),
          const SizedBox(height: 4),
          _AvailabilityBadge(available: installed),
          const SizedBox(height: 24),
          Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [
                  AppColors.blue.withValues(alpha: 0.08),
                  (installed ? AppColors.green : c.textTertiary).withValues(
                    alpha: 0.08,
                  ),
                ],
              ),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: AppColors.blue.withValues(alpha: 0.2)),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  installed ? '本地引擎已就绪 · $acceleration' : '未检测到 whisper.cpp',
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: installed ? AppColors.green : c.textPrimary,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  '当前模式: $whisperMode',
                  style: TextStyle(fontSize: 12, color: c.textSecondary),
                ),
                if (version.isNotEmpty) ...[
                  const SizedBox(height: 3),
                  Text(
                    version,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(fontSize: 11, color: c.textTertiary),
                  ),
                ],
                if (!installed) ...[
                  const SizedBox(height: 14),
                  AppButton(
                    onPressed: installing ? null : onInstall,
                    child: Text(installing ? '安装中…' : '安装本地引擎'),
                  ),
                ],
                if (installMessage.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    installMessage,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(fontSize: 11, color: c.textSecondary),
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 20),
          Text(
            '加速方式',
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w600,
              color: c.textPrimary,
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              _AccelCard(
                title: '自动（推荐）',
                desc: installed ? '实际使用 $acceleration' : '安装后自动检测',
                selected: true,
              ),
              const SizedBox(width: 12),
              const _AccelCard(
                title: 'CPU 兼容模式',
                desc: 'GPU 初始化失败时启用',
                selected: false,
              ),
            ],
          ),
          const SizedBox(height: 24),
          Text(
            '> 检测详情',
            style: TextStyle(fontSize: 13, color: c.textSecondary),
          ),
        ],
      ),
    );
  }
}

class _AccelCard extends StatelessWidget {
  final String title;
  final String desc;
  final bool selected;
  const _AccelCard({
    required this.title,
    required this.desc,
    required this.selected,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Expanded(
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(
          color: selected
              ? AppColors.blue.withValues(alpha: 0.08)
              : c.surfaceBg,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
            color: selected
                ? AppColors.blue.withValues(alpha: 0.4)
                : c.borderLight,
            width: selected ? 1.5 : 0.5,
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w600,
                color: selected ? AppColors.blue : c.textPrimary,
              ),
            ),
            const SizedBox(height: 2),
            Text(desc, style: TextStyle(fontSize: 11, color: c.textSecondary)),
          ],
        ),
      ),
    );
  }
}

// ---- 右侧引擎详情面板 ----

class _EngineDetail extends StatelessWidget {
  final EngineItem engine;
  final Map<String, dynamic> runtime;
  final bool installing;
  final String installMessage;
  final VoidCallback onInstall;

  const _EngineDetail({
    required this.engine,
    required this.runtime,
    required this.installing,
    required this.installMessage,
    required this.onInstall,
  });

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(engine.icon, size: 24, color: engine.color),
              const SizedBox(width: 12),
              Text(
                engine.name,
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w700,
                  color: c.textPrimary,
                ),
              ),
              const SizedBox(width: 12),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: engine.available
                      ? AppColors.green.withValues(alpha: 0.15)
                      : c.textTertiary.withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  engine.available ? '可用' : '未安装',
                  style: TextStyle(
                    fontSize: 10,
                    fontWeight: FontWeight.w600,
                    color: engine.available ? AppColors.green : c.textTertiary,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          if (engine.description.isNotEmpty)
            Text(
              engine.description,
              style: TextStyle(
                fontSize: 13,
                color: c.textSecondary,
                height: 1.5,
              ),
            ),
          const SizedBox(height: 24),
          if (!engine.available) ...[
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: c.surfaceBg,
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: c.borderLight, width: 0.5),
              ),
              child: Center(
                child: Column(
                  children: [
                    Icon(
                      CupertinoIcons.cloud_download,
                      size: 32,
                      color: c.textTertiary,
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '该引擎尚未安装',
                      style: TextStyle(fontSize: 14, color: c.textPrimary),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      '安装后可在此配置和使用',
                      style: TextStyle(fontSize: 12, color: c.textTertiary),
                    ),
                    if (engine.id == 'whisper_cpp') ...[
                      const SizedBox(height: 14),
                      AppButton(
                        onPressed: installing ? null : onInstall,
                        child: Text(installing ? '安装中…' : '安装'),
                      ),
                      if (installMessage.isNotEmpty) ...[
                        const SizedBox(height: 10),
                        Text(
                          installMessage,
                          maxLines: 3,
                          overflow: TextOverflow.ellipsis,
                          textAlign: TextAlign.center,
                          style: TextStyle(
                            fontSize: 11,
                            color: c.textSecondary,
                          ),
                        ),
                      ],
                    ],
                  ],
                ),
              ),
            ),
          ],
          if (engine.id == 'whisper_cpp' && engine.available) ...[
            const SizedBox(height: 8),
            _RuntimeDetails(runtime: runtime),
          ],
        ],
      ),
    );
  }
}

class _AvailabilityBadge extends StatelessWidget {
  final bool available;

  const _AvailabilityBadge({required this.available});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: (available ? AppColors.green : c.textTertiary).withValues(
          alpha: 0.15,
        ),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        available ? '可用' : '未安装',
        style: TextStyle(
          fontSize: 10,
          fontWeight: FontWeight.w600,
          color: available ? AppColors.green : c.textTertiary,
        ),
      ),
    );
  }
}

class _RuntimeDetails extends StatelessWidget {
  final Map<String, dynamic> runtime;

  const _RuntimeDetails({required this.runtime});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    final rows = <(String, String)>[
      ('加速', runtime['acceleration'] as String? ?? 'CPU'),
      ('版本', runtime['version'] as String? ?? ''),
      ('路径', runtime['binary'] as String? ?? ''),
    ];
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: c.surfaceBg,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: c.borderLight, width: 0.5),
      ),
      child: Column(
        children: rows
            .where((row) => row.$2.isNotEmpty)
            .map(
              (row) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 4),
                child: Row(
                  children: [
                    SizedBox(
                      width: 48,
                      child: Text(
                        row.$1,
                        style: TextStyle(fontSize: 12, color: c.textTertiary),
                      ),
                    ),
                    Expanded(
                      child: Text(
                        row.$2,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(fontSize: 12, color: c.textSecondary),
                      ),
                    ),
                  ],
                ),
              ),
            )
            .toList(),
      ),
    );
  }
}
