import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart'
    show
        CupertinoAlertDialog,
        CupertinoDialogAction,
        CupertinoIcons,
        CupertinoPageRoute,
        CupertinoPageScaffold,
        showCupertinoDialog;
import 'package:flutter/widgets.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';
import 'package:macos_ui/macos_ui.dart';

import '../../../../data/models/task.dart';
import '../../../../providers.dart';
import '../../../core/app_button.dart';
import '../../../core/app_colors.dart';
import '../../../core/desktop_dropdown.dart';
import '../../task/view_models/task_list_view_model.dart';
import '../view_models/dashboard_view_model.dart';

class DashboardPage extends HookConsumerWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final backend = ref.watch(backendProvider);
    useListenable(backend);
    final ds = ref.watch(dashboardProvider);

    useEffect(() {
      Future.microtask(() => ref.read(dashboardProvider.notifier).load());
      return null;
    }, const []);

    if (ds.loading && !backend.online) {
      return const Center(child: ProgressCircle(radius: 14));
    }

    return ListView(
      padding: const EdgeInsets.fromLTRB(32, 28, 32, 32),
      children: [
        const _Greeting(),
        const SizedBox(height: 24),
        _WorkflowGrid(ref: ref),
      ],
    );
  }
}

// ---- 问候区域 ----

class _Greeting extends StatelessWidget {
  const _Greeting();

  String get _greetingText {
    final hour = DateTime.now().hour;
    if (hour < 6) return '夜深了';
    if (hour < 12) return '早上好';
    if (hour < 14) return '中午好';
    if (hour < 18) return '下午好';
    return '晚上好';
  }

  String get _dateText {
    final now = DateTime.now();
    const weekdays = ['一', '二', '三', '四', '五', '六', '日'];
    return '${now.month}月${now.day}日星期${weekdays[now.weekday - 1]} · 选择一个任务开始，文件也可以直接拖到卡片上';
  }

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                _greetingText,
                style: TextStyle(
                  fontSize: 26,
                  fontWeight: FontWeight.w800,
                  color: c.textPrimary,
                  letterSpacing: -0.8,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        Text(_dateText, style: TextStyle(fontSize: 13, color: c.textSecondary)),
      ],
    );
  }
}

// ---- 工作流卡片网格 ----

class _WorkflowGrid extends StatelessWidget {
  final WidgetRef ref;
  const _WorkflowGrid({required this.ref});

  @override
  Widget build(BuildContext context) {
    final c = AppColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(bottom: 12),
          child: Text(
            '开始创作',
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w600,
              color: c.textPrimary,
            ),
          ),
        ),
        _buildGrid(context),
      ],
    );
  }

  Widget _buildGrid(BuildContext context) {
    final savedRecipes = ref.watch(
      dashboardProvider.select((state) => state.customWorkflows),
    );
    final workflows = [
      ..._allWorkflows,
      ...savedRecipes.map(
        (recipe) => WorkflowTemplate(
          id: 'recipe_${recipe.name}',
          title: recipe.name,
          subtitle: _recipeSubtitle(recipe),
          gradientColors: const ['007AFF', '5AC8FA'],
          steps: _stepsForMode(recipe.mode),
          mode: recipe.mode,
          recipe: recipe,
        ),
      ),
    ];
    final rows = <Widget>[];
    for (var i = 0; i < workflows.length; i += 3) {
      final end = (i + 3).clamp(0, workflows.length);
      final chunk = workflows.sublist(i, end);
      rows.add(
        Padding(
          padding: EdgeInsets.only(top: i == 0 ? 0 : 12),
          child: Row(
            children: chunk.asMap().entries.map((e) {
              return Expanded(
                child: Padding(
                  padding: EdgeInsets.only(
                    left: e.key == 0 ? 0 : 6,
                    right: e.key == chunk.length - 1 ? 0 : 6,
                  ),
                  child: _LaunchCard(
                    wf: e.value,
                    onTap: () => _handleTap(context, e.value),
                  ),
                ),
              );
            }).toList(),
          ),
        ),
      );
    }
    return Column(children: rows);
  }

  void _handleTap(BuildContext context, WorkflowTemplate wf) {
    Navigator.of(context).push(
      CupertinoPageRoute(builder: (_) => _WorkflowDetailPage(workflow: wf)),
    );
  }

  static String _recipeSubtitle(TaskRecipe recipe) {
    final subtitle = switch (recipe.subtitleOutput) {
      'file' => '独立字幕',
      'none' => '无字幕',
      _ => '烧录字幕',
    };
    return '${recipe.translator} · ${recipe.ttsEngine} · $subtitle';
  }

  static List<WorkflowStep> _stepsForMode(int mode) {
    return [
      const WorkflowStep(id: 'whisper', label: '听写', icon: 'waveform'),
      if (mode >= 2)
        const WorkflowStep(id: 'translate', label: '翻译', icon: 'globe'),
      if (mode >= 3) ...[
        const WorkflowStep(id: 'tts', label: '配音', icon: 'mic'),
        const WorkflowStep(id: 'merge', label: '合成', icon: 'film'),
      ],
    ];
  }

  static final _allWorkflows = <WorkflowTemplate>[
    const WorkflowTemplate(
      id: 'dub_full',
      title: '视频 → 配音成片',
      subtitle: '听写、翻译、配音、合成一条龙，全自动出成品视频',
      gradientColors: ['FF3B30', 'FF6B5B'],
      mode: 3,
      steps: [
        WorkflowStep(id: 'whisper', label: '提取字幕', icon: 'text'),
        WorkflowStep(id: 'translate', label: '翻译字幕', icon: 'globe'),
        WorkflowStep(id: 'tts', label: '语音合成', icon: 'mic'),
        WorkflowStep(id: 'merge', label: '合成视频', icon: 'film'),
      ],
    ),
    const WorkflowTemplate(
      id: 'bilingual_sub',
      title: '视频 → 双语字幕',
      subtitle: '转写人声并翻译成目标语言，一步到位',
      gradientColors: ['30D158', '5BDE83'],
      mode: 2,
      steps: [
        WorkflowStep(id: 'whisper', label: '语音识别', icon: 'waveform'),
        WorkflowStep(id: 'translate', label: '翻译字幕', icon: 'globe'),
      ],
    ),
    const WorkflowTemplate(
      id: 'subtitle_only',
      title: '视频 → 原文字幕',
      subtitle: '只转写人声，不翻译',
      gradientColors: ['007AFF', '5AC8FA'],
      mode: 1,
      steps: [
        WorkflowStep(id: 'whisper', label: '语音识别', icon: 'waveform'),
        WorkflowStep(id: 'output', label: '输出字幕', icon: 'doc'),
      ],
    ),
    const WorkflowTemplate(
      id: 'translate_existing',
      title: '翻译已有字幕',
      subtitle: '把 SRT 等字幕文件翻译成其它语言',
      gradientColors: ['5856D6', '8E8AFF'],
      mode: 4,
      steps: [
        WorkflowStep(id: 'load', label: '加载字幕', icon: 'doc'),
        WorkflowStep(id: 'translate', label: '翻译', icon: 'globe'),
      ],
    ),
    const WorkflowTemplate(
      id: 'en_to_cn',
      title: '英文视频转中文',
      subtitle: '转写 · 翻译 · 配音 · 合成',
      gradientColors: ['FF9500', 'FFCC00'],
      mode: 3,
      steps: [
        WorkflowStep(id: 'whisper', label: '语音识别', icon: 'waveform'),
        WorkflowStep(id: 'translate', label: '翻译', icon: 'globe'),
        WorkflowStep(id: 'tts', label: '配音', icon: 'mic'),
        WorkflowStep(id: 'merge', label: '合成', icon: 'film'),
      ],
    ),
    const WorkflowTemplate(
      id: 'custom',
      title: '自定义流程',
      subtitle: '打开向导，自由组合目标与配置，可存为配方',
      gradientColors: ['8E8E93', 'AEAEB2'],
      mode: 0,
      steps: [],
    ),
  ];
}

// ---- 单张启动卡片 ----

class _LaunchCard extends HookWidget {
  final WorkflowTemplate wf;
  final VoidCallback onTap;
  const _LaunchCard({required this.wf, required this.onTap});

  IconData get _icon {
    switch (wf.id) {
      case 'dub_full':
        return CupertinoIcons.film;
      case 'bilingual_sub':
        return CupertinoIcons.doc_text;
      case 'subtitle_only':
        return CupertinoIcons.textformat;
      case 'translate_existing':
        return CupertinoIcons.doc_on_doc;
      case 'en_to_cn':
        return CupertinoIcons.globe;
      case 'custom':
        return CupertinoIcons.plus;
      default:
        return CupertinoIcons.play;
    }
  }

  @override
  Widget build(BuildContext context) {
    final hovering = useState(false);
    final c = AppColors.of(context);
    final color = Color(int.parse('FF${wf.gradientColors[0]}', radix: 16));

    return MouseRegion(
      onEnter: (_) => hovering.value = true,
      onExit: (_) => hovering.value = false,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          padding: const EdgeInsets.all(16),
          height: 120,
          decoration: BoxDecoration(
            color: hovering.value ? c.cardBgHover : c.cardBg,
            borderRadius: BorderRadius.circular(14),
            border: Border.all(
              color: hovering.value ? color.withValues(alpha: 0.3) : c.border,
              width: 0.5,
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 32,
                height: 32,
                decoration: BoxDecoration(
                  color: color.withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(_icon, size: 16, color: color),
              ),
              const Spacer(),
              Text(
                wf.title,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: c.textPrimary,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                wf.subtitle,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(fontSize: 11, color: c.textSecondary),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ── 工作流详情页（SmartSub 完整向导：配置栏 + 目标产物 + 步骤链 + 文件导入 + 日志） ──

// ignore: unused_element
class _WorkflowDetailPage extends HookConsumerWidget {
  final WorkflowTemplate workflow;
  const _WorkflowDetailPage({required this.workflow});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final c = AppColors.of(context);
    final creating = useState(false);
    final wf = workflow;
    final color = Color(int.parse('FF${wf.gradientColors[0]}', radix: 16));
    final importedFiles = useState<List<String>>([]);

    final ds = ref.watch(dashboardProvider);
    final cfg = ds.config;
    final saved = wf.recipe;
    final videoLang = useState(saved?.sourceLang ?? 'auto');
    final targetLang = useState(
      saved?.targetLang ?? cfg?.pipeline.defaultTargetLang ?? 'zh-CN',
    );
    final translateService = useState(
      saved?.translator ?? cfg?.llm.provider ?? 'bing',
    );
    final outputContent = useState(saved?.outputContent ?? 'bilingual');
    final ttsEngine = useState(saved?.ttsEngine ?? cfg?.tts.type ?? 'edge');
    final ttsVoice = useState(
      saved?.ttsVoice ??
          (cfg?.tts.voice.isNotEmpty == true
              ? cfg!.tts.voice
              : 'zh-CN-XiaoxiaoNeural'),
    );
    final speechRate = useState(saved?.speechRate ?? 1.0);
    final subtitleOutput = useState(
      saved?.subtitleOutput ?? cfg?.pipeline.subtitleOutput ?? 'burn',
    );

    final initialMode = saved?.mode ?? (wf.mode == 0 ? 3 : wf.mode);
    final doTranslate = useState(initialMode >= 2);
    final doDub = useState(initialMode >= 3);
    final doVideo = useState(initialMode >= 3);

    List<String> activeSteps() {
      final steps = <String>['语音识别'];
      if (doTranslate.value) steps.add('翻译字幕');
      if (doDub.value) steps.add('语音合成');
      if (doVideo.value) steps.add('合成视频');
      if (steps.length == 1) steps.add('输出字幕');
      return steps;
    }

    int resolvedMode() {
      if (doVideo.value && doDub.value) return 3;
      if (doTranslate.value) return 2;
      return 1;
    }

    TaskRecipe currentRecipe([String? name]) {
      return TaskRecipe(
        name: name ?? saved?.name ?? '',
        mode: resolvedMode(),
        sourceLang: videoLang.value,
        targetLang: targetLang.value,
        translator: translateService.value,
        outputContent: outputContent.value,
        ttsEngine: ttsEngine.value,
        ttsVoice: ttsVoice.value,
        speechRate: speechRate.value,
        subtitleOutput: subtitleOutput.value,
      );
    }

    Future<void> saveRecipe() async {
      final controller = TextEditingController(text: saved?.name ?? wf.title);
      final name = await showCupertinoDialog<String>(
        context: context,
        builder: (dialogContext) => CupertinoAlertDialog(
          title: const Text('保存配方'),
          content: Padding(
            padding: const EdgeInsets.only(top: 12),
            child: MacosTextField(controller: controller, placeholder: '配方名称'),
          ),
          actions: [
            CupertinoDialogAction(
              onPressed: () => Navigator.pop(dialogContext),
              child: const Text('取消'),
            ),
            CupertinoDialogAction(
              isDefaultAction: true,
              onPressed: () =>
                  Navigator.pop(dialogContext, controller.text.trim()),
              child: const Text('保存'),
            ),
          ],
        ),
      );
      controller.dispose();
      if (name == null || name.isEmpty) return;
      await ref
          .read(dashboardProvider.notifier)
          .saveCustomWorkflow(currentRecipe(name));
    }

    Future<void> pickFiles() async {
      try {
        final result = await FilePicker.platform.pickFiles(
          type: FileType.custom,
          allowedExtensions: [
            'mp4',
            'mkv',
            'mov',
            'avi',
            'webm',
            'mp3',
            'wav',
            'flac',
            'aac',
            'm4a',
            'srt',
            'ass',
          ],
          allowMultiple: true,
        );
        if (result == null || result.files.isEmpty) return;
        final paths = result.files
            .where((f) => f.path != null)
            .map((f) => f.path!)
            .toList();
        importedFiles.value = [...importedFiles.value, ...paths];
      } catch (_) {}
    }

    Future<void> startTask() async {
      if (importedFiles.value.isEmpty) {
        await pickFiles();
        if (importedFiles.value.isEmpty) return;
      }
      creating.value = true;
      try {
        final notifier = ref.read(taskListProvider.notifier);
        for (final path in importedFiles.value) {
          await notifier.createRecipeTask(
            inputPath: path,
            recipe: currentRecipe(),
          );
        }
        if (context.mounted) Navigator.pop(context);
      } catch (e) {
        if (context.mounted) {
          showCupertinoDialog(
            context: context,
            builder: (_) => CupertinoAlertDialog(
              content: Text('创建失败: $e'),
              actions: [
                CupertinoDialogAction(
                  child: const Text('确定'),
                  onPressed: () => Navigator.pop(context),
                ),
              ],
            ),
          );
        }
      }
      creating.value = false;
    }

    final steps = activeSteps();

    return CupertinoPageScaffold(
      backgroundColor: c.contentBg,
      child: Column(
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(8, 8, 12, 6),
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(color: c.borderLight, width: 0.5),
              ),
            ),
            child: Row(
              children: [
                MacosIconButton(
                  icon: Icon(
                    CupertinoIcons.back,
                    size: 18,
                    color: c.textSecondary,
                  ),
                  onPressed: () => Navigator.pop(context),
                  padding: const EdgeInsets.all(6),
                ),
                const SizedBox(width: 2),
                Text(
                  wf.title,
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    color: c.textPrimary,
                  ),
                ),
                const SizedBox(width: 8),
                Flexible(
                  child: Text(
                    '新任务 · 导入文件后自动显示在任务列表',
                    style: TextStyle(fontSize: 11, color: c.textTertiary),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                const Spacer(),
                _SmallButton(
                  icon: CupertinoIcons.square_arrow_down,
                  label: '保存配方',
                  c: c,
                  onTap: saveRecipe,
                ),
                const SizedBox(width: 6),
                _SmallButton(
                  icon: CupertinoIcons.doc_fill,
                  label: '导入文件',
                  c: c,
                  onTap: pickFiles,
                ),
              ],
            ),
          ),

          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
            decoration: BoxDecoration(
              border: Border(
                bottom: BorderSide(color: c.borderLight, width: 0.5),
              ),
            ),
            child: Column(
              children: [
                Wrap(
                  spacing: 12,
                  runSpacing: 8,
                  children: [
                    _ConfigDropdown(
                      label: '源语言',
                      value: _langDisplayName(videoLang.value),
                      icon: CupertinoIcons.globe,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['自动检测', '英语', '日语', '韩语', '法语', '德语'],
                      onChanged: (v) =>
                          videoLang.value = _langCode(v ?? '自动检测'),
                    ),
                    _ConfigDropdown(
                      label: '翻译成',
                      value: _langDisplayName(targetLang.value),
                      icon: CupertinoIcons.arrow_right_arrow_left,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['中文', '英语', '日语', '韩语', '法语', '德语'],
                      onChanged: (v) => targetLang.value = _langCode(v ?? '中文'),
                    ),
                    _ConfigDropdown(
                      label: '翻译服务',
                      value: _translatorName(translateService.value),
                      icon: CupertinoIcons.text_bubble,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['必应', 'DeepLX', 'OpenAI'],
                      onChanged: (v) =>
                          translateService.value = _translatorCode(v ?? '必应'),
                    ),
                    _ConfigDropdown(
                      label: '输出内容',
                      value: _contentName(outputContent.value),
                      icon: CupertinoIcons.doc_text,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['双语字幕', '仅译文', '仅原文'],
                      onChanged: (v) =>
                          outputContent.value = _contentCode(v ?? '双语字幕'),
                    ),
                    _ConfigDropdown(
                      label: '配音引擎',
                      value: ttsEngine.value == 'openai'
                          ? 'OpenAI TTS'
                          : 'Edge TTS',
                      icon: CupertinoIcons.mic,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['Edge TTS', 'OpenAI TTS'],
                      onChanged: (v) => ttsEngine.value = v == 'OpenAI TTS'
                          ? 'openai'
                          : 'edge',
                    ),
                    _ConfigDropdown(
                      label: '音色',
                      value: ttsVoice.value,
                      icon: CupertinoIcons.person_crop_circle,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const [
                        'zh-CN-XiaoxiaoNeural',
                        'zh-CN-XiaoyiNeural',
                        'zh-CN-YunjianNeural',
                        'alloy',
                      ],
                      onChanged: (v) => ttsVoice.value = v ?? ttsVoice.value,
                    ),
                    _ConfigDropdown(
                      label: '语速',
                      value: '${speechRate.value.toStringAsFixed(1)}×',
                      icon: CupertinoIcons.speedometer,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['0.8×', '1.0×', '1.1×', '1.2×'],
                      onChanged: (v) => speechRate.value =
                          double.tryParse((v ?? '1.0×').replaceAll('×', '')) ??
                          1,
                    ),
                    _ConfigDropdown(
                      label: '字幕方式',
                      value: _subtitleModeName(subtitleOutput.value),
                      icon: CupertinoIcons.rectangle_badge_checkmark,
                      iconColor: c.textSecondary,
                      c: c,
                      items: const ['烧录', '独立文件', '无字幕仅配音'],
                      onChanged: (v) =>
                          subtitleOutput.value = _subtitleModeCode(v ?? '烧录'),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                // 目标产物 toggle + 步骤链
                Row(
                  children: [
                    _GoalToggle(
                      label: '翻译',
                      icon: CupertinoIcons.globe,
                      active: doTranslate.value,
                      color: const Color(0xFF30D158),
                      c: c,
                      onTap: () {
                        doTranslate.value = !doTranslate.value;
                        if (!doTranslate.value) {
                          doDub.value = false;
                          doVideo.value = false;
                        }
                      },
                    ),
                    const SizedBox(width: 8),
                    _GoalToggle(
                      label: '配音',
                      icon: CupertinoIcons.mic,
                      active: doDub.value,
                      color: const Color(0xFFFF9500),
                      c: c,
                      onTap: () {
                        doDub.value = !doDub.value;
                        if (doDub.value && !doTranslate.value) {
                          doTranslate.value = true;
                        }
                        if (!doDub.value) {
                          doVideo.value = false;
                        }
                      },
                    ),
                    const SizedBox(width: 8),
                    _GoalToggle(
                      label: '成片',
                      icon: CupertinoIcons.film,
                      active: doVideo.value,
                      color: const Color(0xFFFF3B30),
                      c: c,
                      onTap: () {
                        doVideo.value = !doVideo.value;
                        if (doVideo.value && !doDub.value) {
                          doDub.value = true;
                          doTranslate.value = true;
                        }
                      },
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: SingleChildScrollView(
                        scrollDirection: Axis.horizontal,
                        child: Row(
                          children: steps.asMap().entries.expand((e) {
                            final widgets = <Widget>[
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 10,
                                  vertical: 4,
                                ),
                                decoration: BoxDecoration(
                                  color: color.withValues(
                                    alpha: c.isDark ? 0.12 : 0.08,
                                  ),
                                  borderRadius: BorderRadius.circular(6),
                                ),
                                child: Text(
                                  e.value,
                                  style: TextStyle(
                                    fontSize: 12,
                                    fontWeight: FontWeight.w500,
                                    color: color,
                                  ),
                                ),
                              ),
                            ];
                            if (e.key < steps.length - 1) {
                              widgets.add(
                                Padding(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 4,
                                  ),
                                  child: Icon(
                                    CupertinoIcons.chevron_right,
                                    size: 10,
                                    color: c.textTertiary,
                                  ),
                                ),
                              );
                            }
                            return widgets;
                          }).toList(),
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),

          Expanded(
            child: importedFiles.value.isEmpty
                ? _EmptyImportArea(color: color, c: c, onImport: pickFiles)
                : _ImportedFileList(
                    files: importedFiles.value,
                    c: c,
                    onRemove: (i) {
                      final updated = [...importedFiles.value]..removeAt(i);
                      importedFiles.value = updated;
                    },
                  ),
          ),

          Container(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
            decoration: BoxDecoration(
              border: Border(top: BorderSide(color: c.borderLight, width: 0.5)),
            ),
            child: Row(
              children: [
                Icon(CupertinoIcons.link, size: 13, color: color),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    steps.join('  →  '),
                    style: TextStyle(fontSize: 11, color: c.textSecondary),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (importedFiles.value.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(right: 10),
                    child: Text(
                      '${importedFiles.value.length} 个文件',
                      style: TextStyle(fontSize: 11, color: c.textSecondary),
                    ),
                  ),
                AppButton(
                  onPressed: creating.value ? null : startTask,
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(CupertinoIcons.play_fill, size: 12),
                      const SizedBox(width: 6),
                      Text(
                        creating.value ? '创建中...' : '开始任务',
                        style: const TextStyle(fontWeight: FontWeight.w600),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  static String _langDisplayName(String code) {
    const map = {
      'auto': '自动检测',
      'zh-CN': '中文',
      'en': '英语',
      'ja': '日语',
      'ko': '韩语',
      'fr': '法语',
      'de': '德语',
    };
    return map[code] ?? code;
  }

  static String _langCode(String name) {
    const map = {
      '自动检测': 'auto',
      '中文': 'zh-CN',
      '英语': 'en',
      '日语': 'ja',
      '韩语': 'ko',
      '法语': 'fr',
      '德语': 'de',
    };
    return map[name] ?? name;
  }

  static String _translatorName(String code) {
    return switch (code) {
      'deeplx' => 'DeepLX',
      'openai' => 'OpenAI',
      _ => '必应',
    };
  }

  static String _translatorCode(String name) {
    return switch (name) {
      'DeepLX' => 'deeplx',
      'OpenAI' => 'openai',
      _ => 'bing',
    };
  }

  static String _contentName(String code) {
    return switch (code) {
      'source' => '仅原文',
      'translated' => '仅译文',
      _ => '双语字幕',
    };
  }

  static String _contentCode(String name) {
    return switch (name) {
      '仅原文' => 'source',
      '仅译文' => 'translated',
      _ => 'bilingual',
    };
  }

  static String _subtitleModeName(String code) {
    return switch (code) {
      'file' => '独立文件',
      'none' => '无字幕仅配音',
      _ => '烧录',
    };
  }

  static String _subtitleModeCode(String name) {
    return switch (name) {
      '独立文件' => 'file',
      '无字幕仅配音' => 'none',
      _ => 'burn',
    };
  }
}

/// 空状态文件导入区（虚线边框 + 三步引导）
class _EmptyImportArea extends StatelessWidget {
  final Color color;
  final AppColors c;
  final VoidCallback onImport;
  const _EmptyImportArea({
    required this.color,
    required this.c,
    required this.onImport,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Container(
        decoration: BoxDecoration(
          color: c.cardBg,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: color.withValues(alpha: 0.3), width: 1),
        ),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              _GuideStep(
                index: 1,
                label: '导入文件',
                desc: '支持 MP4 / MKV / MOV / MP3 / WAV 等 29 种常见视频与音频格式',
                color: color,
                c: c,
              ),
              const SizedBox(height: 20),
              _GuideStep(
                index: 2,
                label: '核对配置',
                desc: '在上方选择语言、引擎、音色与输出方式',
                color: color,
                c: c,
              ),
              const SizedBox(height: 20),
              _GuideStep(
                index: 3,
                label: '开始任务',
                desc: '点击右下角开始按钮，批量处理并实时查看进度',
                color: color,
                c: c,
              ),
              const SizedBox(height: 28),
              AppButton(
                onPressed: onImport,
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(CupertinoIcons.doc_fill, size: 14),
                    SizedBox(width: 8),
                    Text('导入', style: TextStyle(fontWeight: FontWeight.w600)),
                  ],
                ),
              ),
              const SizedBox(height: 10),
              Text(
                '也可以把媒体文件或文件夹直接拖进本页',
                style: TextStyle(fontSize: 11, color: c.textTertiary),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// 已导入文件列表
class _ImportedFileList extends StatelessWidget {
  final List<String> files;
  final AppColors c;
  final void Function(int index) onRemove;
  const _ImportedFileList({
    required this.files,
    required this.c,
    required this.onRemove,
  });

  @override
  Widget build(BuildContext context) {
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: files.length,
      itemBuilder: (_, i) {
        final name = files[i].split('/').last;
        return Container(
          margin: const EdgeInsets.only(bottom: 6),
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: c.cardBg,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: c.border, width: 0.5),
          ),
          child: Row(
            children: [
              Icon(CupertinoIcons.film, size: 16, color: c.textSecondary),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  name,
                  style: TextStyle(fontSize: 13, color: c.textPrimary),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              MacosIconButton(
                icon: Icon(
                  CupertinoIcons.xmark_circle_fill,
                  size: 16,
                  color: c.textTertiary,
                ),
                onPressed: () => onRemove(i),
                padding: EdgeInsets.zero,
              ),
            ],
          ),
        );
      },
    );
  }
}

/// 配置下拉选择器 — 桌面风格原地 dropdown overlay
class _ConfigDropdown extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;
  final Color iconColor;
  final AppColors c;
  final List<String> items;
  final ValueChanged<String?> onChanged;

  const _ConfigDropdown({
    required this.label,
    required this.value,
    required this.icon,
    required this.iconColor,
    required this.c,
    required this.items,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    return DesktopDropdown(
      value: value,
      items: items,
      onChanged: (v) => onChanged(v),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(label, style: TextStyle(fontSize: 12, color: c.textSecondary)),
          const SizedBox(width: 6),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
            decoration: BoxDecoration(
              color: c.inputBg,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(icon, size: 12, color: iconColor),
                const SizedBox(width: 5),
                Text(
                  value,
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    color: c.textPrimary,
                  ),
                ),
                const SizedBox(width: 4),
                Icon(
                  CupertinoIcons.chevron_down,
                  size: 10,
                  color: c.textTertiary,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// 目标产物 toggle 按钮
class _GoalToggle extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool active;
  final Color color;
  final AppColors c;
  final VoidCallback onTap;

  const _GoalToggle({
    required this.label,
    required this.icon,
    required this.active,
    required this.color,
    required this.c,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: active ? color.withValues(alpha: 0.12) : c.cardBg,
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
            color: active ? color.withValues(alpha: 0.4) : c.border,
            width: active ? 1.5 : 0.5,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              active ? CupertinoIcons.checkmark_circle_fill : icon,
              size: 14,
              color: active ? color : c.textTertiary,
            ),
            const SizedBox(width: 6),
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: active ? FontWeight.w600 : FontWeight.w400,
                color: active ? color : c.textSecondary,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 引导步骤行
class _GuideStep extends StatelessWidget {
  final int index;
  final String label;
  final String desc;
  final Color color;
  final AppColors c;

  const _GuideStep({
    required this.index,
    required this.label,
    required this.desc,
    required this.color,
    required this.c,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Container(
          width: 24,
          height: 24,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: color.withValues(alpha: 0.12),
          ),
          child: Center(
            child: Text(
              '$index',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w700,
                color: color,
              ),
            ),
          ),
        ),
        const SizedBox(width: 10),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w600,
                color: c.textPrimary,
              ),
            ),
            const SizedBox(height: 2),
            Text(desc, style: TextStyle(fontSize: 12, color: c.textSecondary)),
          ],
        ),
      ],
    );
  }
}

/// 顶栏小按钮
class _SmallButton extends StatelessWidget {
  final IconData icon;
  final String? label;
  final AppColors c;
  final VoidCallback onTap;
  const _SmallButton({
    required this.icon,
    this.label,
    required this.c,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 13, color: c.textSecondary),
              if (label != null) ...[
                const SizedBox(width: 4),
                Text(
                  label!,
                  style: TextStyle(fontSize: 11, color: c.textSecondary),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
