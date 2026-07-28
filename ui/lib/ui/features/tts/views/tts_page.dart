import 'dart:ui';
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../core/app_colors.dart';
import '../view_models/tts_view_model.dart';

/// 音色（TTS 服务）配置页
class TTSPage extends HookConsumerWidget {
  const TTSPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ts = ref.watch(ttsProvider);
    final notifier = ref.read(ttsProvider.notifier);

    useEffect(() {
      Future.microtask(() => notifier.load());
      return null;
    }, const []);

    return Row(
      children: [
        _ServiceList(
          services: ts.services,
          selectedId: ts.selectedServiceId,
          onSelect: notifier.selectService,
        ),
        Expanded(
          child: ts.selectedServiceId == null
              ? const _OverviewPanel()
              : _ServiceDetail(
                  service: ts.services.firstWhere(
                    (s) => s.id == ts.selectedServiceId,
                    orElse: () => TTSService.empty,
                  ),
                  notifier: notifier,
                ),
        ),
      ],
    );
  }
}

// ---- 左侧服务列表 ----

class _ServiceList extends HookWidget {
  final List<TTSService> services;
  final String? selectedId;
  final ValueChanged<String?> onSelect;

  const _ServiceList({
    required this.services,
    required this.selectedId,
    required this.onSelect,
  });

  @override
  Widget build(BuildContext context) {
    final local = services.where((s) => s.category == 'local').toList();
    final online = services.where((s) => s.category == 'online').toList();

    final c = AppColors.of(context);
    return ClipRect(
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 30, sigmaY: 30),
        child: Container(
          width: 260,
          decoration: BoxDecoration(
            color: c.barBg.withValues(alpha: 0.9),
            border: Border(
                right: BorderSide(color: c.borderLight, width: 0.5)),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
                child: Row(
                  children: [
                    Text('配音声音',
                        style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                            color: c.textPrimary)),
                    const Spacer(),
                    Icon(CupertinoIcons.plus,
                        size: 16, color: c.textSecondary),
                  ],
                ),
              ),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  children: [
                    if (local.isNotEmpty) ...[
                      _sectionTitle('本地模型', c),
                      ...local.map((s) => _ServiceItem(
                            service: s,
                            selected: s.id == selectedId,
                            onTap: () => onSelect(s.id),
                          )),
                    ],
                    if (online.isNotEmpty) ...[
                      _sectionTitle('在线服务', c),
                      ...online.map((s) => _ServiceItem(
                            service: s,
                            selected: s.id == selectedId,
                            onTap: () => onSelect(s.id),
                          )),
                    ],
                    const SizedBox(height: 12),
                    GestureDetector(
                      onTap: () {},
                      child: Container(
                        margin: const EdgeInsets.symmetric(horizontal: 4),
                        padding: const EdgeInsets.symmetric(
                            horizontal: 12, vertical: 10),
                        decoration: BoxDecoration(
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(
                              color: c.borderLight, width: 0.5),
                        ),
                        child: Row(
                          children: [
                            Icon(CupertinoIcons.plus,
                                size: 14, color: c.textTertiary),
                            const SizedBox(width: 8),
                            Text('添加自定义服务',
                                style: TextStyle(
                                    fontSize: 12, color: c.textTertiary)),
                          ],
                        ),
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

  Widget _sectionTitle(String title, AppColors c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: Text(title,
          style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w600,
              color: c.textTertiary)),
    );
  }
}

class _ServiceItem extends HookWidget {
  final TTSService service;
  final bool selected;
  final VoidCallback onTap;
  const _ServiceItem(
      {required this.service, required this.selected, required this.onTap});

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
              Icon(service.icon, size: 16, color: service.color),
              const SizedBox(width: 10),
              Expanded(
                child: Text(service.name,
                    style: TextStyle(
                        fontSize: 13,
                        fontWeight:
                            selected ? FontWeight.w600 : FontWeight.w400,
                        color: selected
                            ? AppColors.blue
                            : c.textPrimary)),
              ),
              if (service.badge.isNotEmpty) ...[
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 5, vertical: 2),
                  decoration: BoxDecoration(
                    color: AppColors.orange.withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(service.badge,
                      style: const TextStyle(
                          fontSize: 9,
                          fontWeight: FontWeight.w600,
                          color: AppColors.orange)),
                ),
                const SizedBox(width: 6),
              ],
              Container(
                width: 7,
                height: 7,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: service.available
                      ? AppColors.green
                      : c.borderSubtle,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---- 右侧总览 ----

class _OverviewPanel extends StatelessWidget {
  const _OverviewPanel();

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('配音声音 — 总览',
              style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w700,
                  color: c.textPrimary)),
          const SizedBox(height: 8),
          Text('三类声音来源一览与推荐起步路径',
              style: TextStyle(fontSize: 13, color: c.textSecondary)),
          const SizedBox(height: 24),
          _OverviewCard(
            title: '本地模型',
            desc: 'Kokoro / VITS / ZipVoice 等，无需网络，隐私安全',
            icon: CupertinoIcons.desktopcomputer,
          ),
          const SizedBox(height: 12),
          _OverviewCard(
            title: '在线服务',
            desc: 'OpenAI / Azure / Edge TTS / ElevenLabs 等云端服务',
            icon: CupertinoIcons.cloud,
          ),
        ],
      ),
    );
  }
}

class _OverviewCard extends StatelessWidget {
  final String title;
  final String desc;
  final IconData icon;
  const _OverviewCard(
      {required this.title, required this.desc, required this.icon});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: c.surfaceBg,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: c.borderLight, width: 0.5),
      ),
      child: Row(
        children: [
          Icon(icon, size: 20, color: AppColors.blue),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                        color: c.textPrimary)),
                const SizedBox(height: 3),
                Text(desc,
                    style: TextStyle(
                        fontSize: 12, color: c.textSecondary)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ---- 右侧服务详情 ----

class _ServiceDetail extends HookWidget {
  final TTSService service;
  final TTSNotifier notifier;
  const _ServiceDetail({required this.service, required this.notifier});

  @override
  Widget build(BuildContext context) {
    final testing = useState(false);
    final c = AppColors.of(context);

    return ListView(
      padding: const EdgeInsets.all(32),
      children: [
        Row(
          children: [
            Icon(service.icon, size: 24, color: service.color),
            const SizedBox(width: 12),
            Text(service.name,
                style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w700,
                    color: c.textPrimary)),
            const SizedBox(width: 12),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: service.available
                    ? AppColors.green.withValues(alpha: 0.15)
                    : c.borderSubtle.withValues(alpha: 0.3),
                borderRadius: BorderRadius.circular(4),
              ),
              child: Text(service.available ? '可用' : '未配置',
                  style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w600,
                      color: service.available
                          ? AppColors.green
                          : c.textTertiary)),
            ),
          ],
        ),
        const SizedBox(height: 16),
        if (service.description.isNotEmpty)
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: AppColors.blue.withValues(alpha: 0.06),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(
                  color: AppColors.blue.withValues(alpha: 0.15)),
            ),
            child: Text(service.description,
                style: TextStyle(
                    fontSize: 12,
                    color: c.textSecondary,
                    height: 1.5)),
          ),
        const SizedBox(height: 20),
        Row(
          children: [
            _ActionButton(
              label: '音色文档',
              icon: CupertinoIcons.doc_text,
              onTap: () {},
            ),
            const SizedBox(width: 8),
            _ActionButton(
              label: '清除配置',
              icon: CupertinoIcons.refresh,
              onTap: () {},
            ),
            const SizedBox(width: 8),
            PushButton(
              controlSize: ControlSize.small,
              secondary: true,
              onPressed: testing.value
                  ? null
                  : () async {
                      testing.value = true;
                      await notifier.testConnection(service.id);
                      testing.value = false;
                    },
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(CupertinoIcons.play_fill,
                      size: 11, color: Color(0xFFFFFFFF)),
                  const SizedBox(width: 5),
                  Text(testing.value ? '测试中...' : '测试连接',
                      style: const TextStyle(
                          fontSize: 12, color: Color(0xFFFFFFFF))),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 24),
        Text('音色候选',
            style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: c.textPrimary)),
        const SizedBox(height: 8),
        Text('微软 Neural 音色名（如 zh-CN-XiaoxiaoNeural），逗号分隔',
            style: TextStyle(fontSize: 11, color: c.textTertiary)),
        const SizedBox(height: 8),
        MacosTextField(
          placeholder: '输入音色名，回车添加',
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: c.inputBg,
            borderRadius: const BorderRadius.all(Radius.circular(10)),
          ),
          style: TextStyle(fontSize: 13, color: c.textPrimary),
          placeholderStyle: TextStyle(fontSize: 13, color: c.textPlaceholder),
        ),
        const SizedBox(height: 20),
        Text('请求超时（秒）',
            style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: c.textPrimary)),
        const SizedBox(height: 8),
        MacosTextField(
          placeholder: '60',
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: c.inputBg,
            borderRadius: const BorderRadius.all(Radius.circular(10)),
          ),
          style: TextStyle(fontSize: 13, color: c.textPrimary),
          placeholderStyle: TextStyle(fontSize: 13, color: c.textPlaceholder),
        ),
        const SizedBox(height: 8),
        Text('单次合成请求超时（秒）',
            style: TextStyle(fontSize: 11, color: c.textTertiary)),
        const SizedBox(height: 20),
        Text('并发数',
            style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: c.textPrimary)),
        const SizedBox(height: 8),
        MacosTextField(
          placeholder: '1',
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: c.inputBg,
            borderRadius: const BorderRadius.all(Radius.circular(10)),
          ),
          style: TextStyle(fontSize: 13, color: c.textPrimary),
          placeholderStyle: TextStyle(fontSize: 13, color: c.textPlaceholder),
        ),
        const SizedBox(height: 8),
        Text('批量合成时的并发请求数（跨任务全局生效）',
            style: TextStyle(fontSize: 11, color: c.textTertiary)),
      ],
    );
  }
}

class _ActionButton extends StatelessWidget {
  final String label;
  final IconData icon;
  final VoidCallback onTap;
  const _ActionButton(
      {required this.label, required this.icon, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: c.inputBg,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(color: c.borderLight, width: 0.5),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: c.textTertiary),
            const SizedBox(width: 5),
            Text(label,
                style: TextStyle(
                    fontSize: 12, color: c.textPrimary)),
          ],
        ),
      ),
    );
  }
}
