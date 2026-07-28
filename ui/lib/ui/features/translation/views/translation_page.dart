import 'dart:ui';
import 'package:flutter/cupertino.dart' show CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../core/app_colors.dart';
import '../view_models/translation_view_model.dart';

/// 翻译服务配置页 — SmartSub 左右分栏风格
class TranslationPage extends HookConsumerWidget {
  const TranslationPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ts = ref.watch(translationProvider);
    final notifier = ref.read(translationProvider.notifier);

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
          showConfiguredOnly: ts.showConfiguredOnly,
          onToggleFilter: notifier.toggleConfiguredFilter,
        ),
        Expanded(
          child: ts.selectedServiceId == null
              ? const _OverviewPanel()
              : _ServiceConfig(
                  service: ts.services.firstWhere(
                    (s) => s.id == ts.selectedServiceId,
                    orElse: () => TranslationService.empty,
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
  final List<TranslationService> services;
  final String? selectedId;
  final ValueChanged<String?> onSelect;
  final bool showConfiguredOnly;
  final VoidCallback onToggleFilter;

  const _ServiceList({
    required this.services,
    required this.selectedId,
    required this.onSelect,
    required this.showConfiguredOnly,
    required this.onToggleFilter,
  });

  @override
  Widget build(BuildContext context) {
    final filtered = showConfiguredOnly
        ? services.where((s) => s.configured).toList()
        : services;

    final custom = filtered.where((s) => s.category == 'custom').toList();
    final free = filtered.where((s) => s.category == 'free').toList();
    final ai = filtered.where((s) => s.category == 'ai').toList();

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
              _buildHeader(c),
              _buildSearch(c),
              _buildFilter(c),
              Expanded(
                child: ListView(
                  padding: const EdgeInsets.symmetric(horizontal: 8),
                  children: [
                    _buildOverviewItem(c),
                    if (custom.isNotEmpty) ...[
                      _sectionTitle('自定义服务商', c),
                      ...custom.map((s) => _ServiceItem(
                            service: s,
                            selected: s.id == selectedId,
                            onTap: () => onSelect(s.id),
                          )),
                    ],
                    if (free.isNotEmpty) ...[
                      _sectionTitle('免费起步', c),
                      ...free.map((s) => _ServiceItem(
                            service: s,
                            selected: s.id == selectedId,
                            onTap: () => onSelect(s.id),
                          )),
                    ],
                    if (ai.isNotEmpty) ...[
                      _sectionTitle('AI 翻译', c),
                      ...ai.map((s) => _ServiceItem(
                            service: s,
                            selected: s.id == selectedId,
                            onTap: () => onSelect(s.id),
                          )),
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

  Widget _buildHeader(AppColors c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 0),
      child: Row(
        children: [
          Text('翻译服务',
              style: TextStyle(
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  color: c.textPrimary)),
          const Spacer(),
          Icon(CupertinoIcons.plus, size: 16, color: c.textTertiary),
        ],
      ),
    );
  }

  Widget _buildSearch(AppColors c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: MacosTextField(
        placeholder: '搜索服务商',
        placeholderStyle: TextStyle(fontSize: 12, color: c.textPlaceholder),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          color: c.inputBg,
          borderRadius: const BorderRadius.all(Radius.circular(8)),
        ),
        style: TextStyle(fontSize: 12, color: c.textPrimary),
        prefix: Padding(
          padding: const EdgeInsets.only(left: 8),
          child: Icon(CupertinoIcons.search, size: 14, color: c.textPlaceholder),
        ),
      ),
    );
  }

  Widget _buildFilter(AppColors c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
      child: Row(
        children: [
          Text('仅显示已配置',
              style: TextStyle(fontSize: 11, color: c.textTertiary)),
          const Spacer(),
          GestureDetector(
            onTap: onToggleFilter,
            child: Container(
              width: 34,
              height: 18,
              decoration: BoxDecoration(
                color: showConfiguredOnly
                    ? AppColors.blue
                    : c.borderSubtle,
                borderRadius: BorderRadius.circular(9),
              ),
              child: AnimatedAlign(
                duration: const Duration(milliseconds: 150),
                alignment: showConfiguredOnly
                    ? Alignment.centerRight
                    : Alignment.centerLeft,
                child: Container(
                  width: 14,
                  height: 14,
                  margin: const EdgeInsets.symmetric(horizontal: 2),
                  decoration: const BoxDecoration(
                    shape: BoxShape.circle,
                    color: Color(0xFFFFFFFF),
                  ),
                ),
              ),
            ),
          ),
        ],
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
            Icon(CupertinoIcons.squares_below_rectangle,
                size: 16, color: c.textTertiary),
            const SizedBox(width: 10),
            Text('总览',
                style: TextStyle(fontSize: 13, color: c.textPrimary)),
            const SizedBox(width: 6),
            Text('配置状态与起步建议',
                style: TextStyle(fontSize: 10, color: c.textTertiary)),
          ],
        ),
      ),
    );
  }

  Widget _sectionTitle(String title, AppColors c) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: Row(
        children: [
          Text(title,
              style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: c.textTertiary)),
          const Spacer(),
          Icon(CupertinoIcons.chevron_down,
              size: 10, color: c.textTertiary),
        ],
      ),
    );
  }
}

class _ServiceItem extends HookWidget {
  final TranslationService service;
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
                        fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                        color: selected
                            ? AppColors.blue
                            : c.textPrimary)),
              ),
              if (service.configured)
                Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: AppColors.green.withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: const Text('已配置',
                      style: TextStyle(
                          fontSize: 9,
                          fontWeight: FontWeight.w600,
                          color: AppColors.green)),
                ),
              const SizedBox(width: 6),
              Container(
                width: 7,
                height: 7,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: service.configured
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

// ---- 右侧总览面板 ----

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
          Text('翻译服务 — 总览',
              style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w700,
                  color: c.textPrimary)),
          const SizedBox(height: 8),
          Text('配置状态与起步建议',
              style: TextStyle(fontSize: 13, color: c.textSecondary)),
          const SizedBox(height: 24),
          _InfoCard(
            icon: CupertinoIcons.lightbulb,
            title: '快速起步',
            content: '选择左侧任一翻译服务并配置 API Key 即可使用。\n推荐 OpenAI 兼容服务作为 AI 翻译主力。',
          ),
          const SizedBox(height: 12),
          _InfoCard(
            icon: CupertinoIcons.shield,
            title: '免费方案',
            content: '必应翻译、DeepLX 均为免费服务，\n适合轻量级使用或作为备用方案。',
          ),
        ],
      ),
    );
  }
}

class _InfoCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String content;
  const _InfoCard(
      {required this.icon, required this.title, required this.content});

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
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: AppColors.blue),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                        color: c.textPrimary)),
                const SizedBox(height: 4),
                Text(content,
                    style: TextStyle(
                        fontSize: 12,
                        color: c.textSecondary,
                        height: 1.5)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ---- 右侧服务配置面板 ----

class _ServiceConfig extends HookConsumerWidget {
  final TranslationService service;
  final TranslationNotifier notifier;
  const _ServiceConfig({required this.service, required this.notifier});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final baseUrlCtrl = useTextEditingController(text: service.baseUrl);
    final apiKeyCtrl = useTextEditingController(text: service.apiKey);
    final modelCtrl = useTextEditingController(text: service.model);
    final testing = useState(false);
    final testResult = useState<String?>(null);
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
            const SizedBox(width: 8),
            Icon(CupertinoIcons.pencil,
                size: 14, color: c.textTertiary),
            const Spacer(),
            PushButton(
              controlSize: ControlSize.small,
              color: AppColors.blue,
              onPressed: testing.value
                  ? null
                  : () async {
                      testing.value = true;
                      testResult.value = null;
                      try {
                        await notifier.testTranslation(service.id);
                        testResult.value = '测试成功';
                      } catch (e) {
                        testResult.value = '测试失败: $e';
                      }
                      testing.value = false;
                    },
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(CupertinoIcons.play_fill,
                      size: 12, color: Color(0xFFFFFFFF)),
                  const SizedBox(width: 5),
                  Text(testing.value ? '测试中...' : '测试翻译',
                      style: const TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                          color: Color(0xFFFFFFFF))),
                ],
              ),
            ),
          ],
        ),
        if (testResult.value != null) ...[
          const SizedBox(height: 8),
          Text(testResult.value!,
              style: TextStyle(
                  fontSize: 12,
                  color: testResult.value!.startsWith('测试成功')
                      ? AppColors.green
                      : AppColors.red)),
        ],
        const SizedBox(height: 28),
        if (service.description.isNotEmpty) ...[
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
          const SizedBox(height: 24),
        ],
        _buildField('Base URL', baseUrlCtrl, 'https://api.openai.com/v1', c),
        const SizedBox(height: 16),
        _buildField('API Key', apiKeyCtrl, 'sk-...', c, obscure: true),
        const SizedBox(height: 16),
        _buildField('模型名称', modelCtrl, '如 gpt-4o-mini', c),
        const SizedBox(height: 24),
        Row(
          children: [
            const Spacer(),
            PushButton(
              controlSize: ControlSize.regular,
              color: AppColors.green,
              onPressed: () {
                notifier.saveServiceConfig(
                  service.id,
                  baseUrl: baseUrlCtrl.text,
                  apiKey: apiKeyCtrl.text,
                  model: modelCtrl.text,
                );
              },
              child: const Text('保存配置',
                  style: TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                      color: Color(0xFFFFFFFF))),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildField(
      String label, TextEditingController ctrl, String placeholder, AppColors c,
      {bool obscure = false}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text(label,
                style: TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                    color: c.textPrimary)),
            const Text(' *',
                style: TextStyle(fontSize: 13, color: AppColors.red)),
          ],
        ),
        const SizedBox(height: 8),
        MacosTextField(
          controller: ctrl,
          placeholder: placeholder,
          obscureText: obscure,
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          decoration: BoxDecoration(
            color: c.inputBg,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: c.borderLight, width: 0.5),
          ),
          style: TextStyle(fontSize: 13, color: c.textPrimary),
          placeholderStyle:
              TextStyle(fontSize: 13, color: c.textPlaceholder),
        ),
      ],
    );
  }
}
