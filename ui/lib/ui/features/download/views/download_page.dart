import 'package:flutter/cupertino.dart' show CupertinoColors, CupertinoIcons;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';

/// 下载页 — 视频下载引擎管理 + 链接下载
class DownloadPage extends HookWidget {
  const DownloadPage({super.key});

  @override
  Widget build(BuildContext context) {
    final urlController = useTextEditingController();

    return ListView(
      padding: const EdgeInsets.all(32),
      children: [
        _EngineSection(),
        const SizedBox(height: 28),
        _DownloadSection(urlController: urlController),
      ],
    );
  }
}

// ---- 安装下载引擎 ----

class _EngineSection extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text('安装下载引擎',
                style: TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w700,
                    color: c.textPrimary)),
            const Spacer(),
            GestureDetector(
              onTap: () {},
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(CupertinoIcons.refresh,
                      size: 13, color: Color(0xFF007AFF)),
                  const SizedBox(width: 4),
                  const Text('检查更新',
                      style: TextStyle(
                          fontSize: 12, color: Color(0xFF007AFF))),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Text(
          '首次使用需安装下载引擎：yt-dlp 覆盖 YouTube 等海外站点，lux 擅长 B站、抖音等国内站点。安装到本机后可随时在此更新。',
          style: TextStyle(fontSize: 12, color: c.textSecondary, height: 1.5),
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(child: _EngineCard(
              name: 'yt-dlp',
              desc: 'YouTube / 1800+ sites',
              installed: false,
            )),
            const SizedBox(width: 12),
            Expanded(child: _EngineCard(
              name: 'lux',
              desc: 'Bilibili / 抖音 / 小红书...',
              installed: false,
            )),
          ],
        ),
      ],
    );
  }
}

class _EngineCard extends HookWidget {
  final String name;
  final String desc;
  final bool installed;
  const _EngineCard(
      {required this.name, required this.desc, required this.installed});

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final c = AppColors.of(context);

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: hovering.value ? c.cardBgHover : c.cardBg,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(color: c.border, width: 0.5),
        ),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text(name,
                          style: TextStyle(
                              fontSize: 14,
                              fontWeight: FontWeight.w600,
                              color: c.textPrimary)),
                      const SizedBox(width: 8),
                      Text(desc,
                          style: TextStyle(
                              fontSize: 11, color: c.textSecondary)),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(installed ? '已安装' : '未安装',
                      style: TextStyle(
                          fontSize: 11,
                          color: installed
                              ? AppColors.green
                              : c.textTertiary)),
                ],
              ),
            ),
            AppButton(
              onPressed: () {},
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const Icon(CupertinoIcons.cloud_download, size: 12),
                  const SizedBox(width: 5),
                  Text(installed ? '更新' : '安装'),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ---- 视频下载 ----

class _DownloadSection extends StatelessWidget {
  final TextEditingController urlController;
  const _DownloadSection({required this.urlController});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('视频下载',
            style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: c.textPrimary)),
        const SizedBox(height: 12),
        Container(
          height: 200,
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: c.cardBg,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: c.border, width: 0.5),
          ),
          child: MacosTextField(
            controller: urlController,
            placeholder: '粘贴视频链接，一行一条...\n支持从聊天记录等混杂文本中自动识别链接',
            maxLines: null,
            padding: EdgeInsets.zero,
            decoration: const BoxDecoration(),
            style: TextStyle(fontSize: 13, color: c.textPrimary),
            placeholderStyle:
                TextStyle(fontSize: 13, color: c.textTertiary),
          ),
        ),
        const SizedBox(height: 16),
        _DownloadOptions(),
        const SizedBox(height: 16),
        Row(
          children: [
            const Text('请先安装下载引擎',
                style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: Color(0xFFFF3B30))),
            const Spacer(),
            AppButton.secondary(
              onPressed: () {},
              child: const Text('直接下载'),
            ),
            const SizedBox(width: 8),
            AppButton(
              onPressed: () {},
              child: const Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(CupertinoIcons.search, size: 13),
                  SizedBox(width: 5),
                  Text('解析链接'),
                ],
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _DownloadOptions extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: BoxDecoration(
        color: c.cardBg,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: c.border, width: 0.5),
      ),
      child: Row(
        children: [
          _optionItem('保存到', CupertinoIcons.folder, '选择目录', c),
          const SizedBox(width: 16),
          _optionItem('清晰度', null, '最佳', c),
          const SizedBox(width: 16),
          _optionItem('引擎', null, '自动（按站点路由）', c),
          const SizedBox(width: 16),
          _optionItem('并发数', null, '2', c),
          const Spacer(),
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 14,
                height: 14,
                decoration: BoxDecoration(
                  color: const Color(0xFF007AFF),
                  borderRadius: BorderRadius.circular(3),
                ),
                child: const Icon(CupertinoIcons.checkmark,
                    size: 10, color: CupertinoColors.white),
              ),
              const SizedBox(width: 5),
              Text('同时下载官方字幕',
                  style: TextStyle(fontSize: 11, color: c.textPrimary)),
            ],
          ),
          const SizedBox(width: 12),
          GestureDetector(
            onTap: () {},
            child: Container(
              padding:
                  const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: c.inputBg,
                borderRadius: BorderRadius.circular(6),
                border:
                    Border.all(color: c.borderLight, width: 0.5),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(CupertinoIcons.globe,
                      size: 11, color: c.textTertiary),
                  const SizedBox(width: 4),
                  Text('站点 Cookie',
                      style: TextStyle(
                          fontSize: 11, color: c.textPrimary)),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _optionItem(String label, IconData? icon, String value, AppColors c) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text('$label ',
            style: TextStyle(fontSize: 11, color: c.textTertiary)),
        if (icon != null) ...[
          Icon(icon, size: 12, color: AppColors.blue),
          const SizedBox(width: 3),
        ],
        Text(value,
            style: TextStyle(fontSize: 11, color: c.textPrimary)),
        Icon(CupertinoIcons.chevron_down,
            size: 9, color: c.textTertiary),
      ],
    );
  }
}
