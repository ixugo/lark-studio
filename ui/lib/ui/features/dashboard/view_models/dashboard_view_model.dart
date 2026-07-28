import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/models/config.dart';
import '../../../../data/models/task.dart';
import '../../../../data/repositories/config_repository.dart';
import '../../../../data/repositories/task_repository.dart';
import '../../../../data/services/local_storage.dart';
import '../../../../providers.dart';

/// 工作流步骤定义
class WorkflowStep {
  final String id;
  final String label;
  final String icon;
  const WorkflowStep({
    required this.id,
    required this.label,
    required this.icon,
  });
}

/// 工作流模板定义
class WorkflowTemplate {
  final String id;
  final String title;
  final String subtitle;
  final List<String> gradientColors;
  final List<WorkflowStep> steps;
  final int mode;
  final TaskRecipe? recipe;

  const WorkflowTemplate({
    required this.id,
    required this.title,
    required this.subtitle,
    required this.gradientColors,
    required this.steps,
    required this.mode,
    this.recipe,
  });
}

/// 内置工作流模板
const builtInWorkflows = <WorkflowTemplate>[
  WorkflowTemplate(
    id: 'dub_full',
    title: '配音成片',
    subtitle: '完整的视频翻译配音工作流',
    gradientColors: ['007AFF', '5AC8FA'],
    mode: 3,
    steps: [
      WorkflowStep(id: 'whisper', label: '提取字幕', icon: 'text_badge_checkmark'),
      WorkflowStep(id: 'translate', label: '翻译字幕', icon: 'globe'),
      WorkflowStep(id: 'tts', label: '语音合成', icon: 'mic'),
      WorkflowStep(id: 'merge', label: '合成视频', icon: 'film'),
    ],
  ),
  WorkflowTemplate(
    id: 'subtitle_only',
    title: '提取字幕',
    subtitle: 'Whisper 语音识别生成字幕文件',
    gradientColors: ['AF52DE', 'DA8FFF'],
    mode: 1,
    steps: [
      WorkflowStep(id: 'whisper', label: '语音识别', icon: 'waveform'),
      WorkflowStep(id: 'burn', label: '输出字幕', icon: 'doc_text'),
    ],
  ),
  WorkflowTemplate(
    id: 'custom',
    title: '自定义流程',
    subtitle: '按需组合处理步骤',
    gradientColors: ['FF9500', 'FFCC00'],
    mode: 0,
    steps: [],
  ),
];

/// 首页仪表盘状态
class DashboardState {
  final AppConfig? config;
  final List<Task> recentTasks;
  final bool loading;
  final List<TaskRecipe> customWorkflows;

  const DashboardState({
    this.config,
    this.recentTasks = const [],
    this.loading = true,
    this.customWorkflows = const [],
  });

  DashboardState copyWith({
    AppConfig? config,
    List<Task>? recentTasks,
    bool? loading,
    List<TaskRecipe>? customWorkflows,
  }) {
    return DashboardState(
      config: config ?? this.config,
      recentTasks: recentTasks ?? this.recentTasks,
      loading: loading ?? this.loading,
      customWorkflows: customWorkflows ?? this.customWorkflows,
    );
  }
}

/// 首页仪表盘状态管理
class DashboardNotifier extends Notifier<DashboardState> {
  late final ConfigRepository _configRepo;
  late final TaskRepository _taskRepo;
  late final LocalStorage _storage;

  @override
  DashboardState build() {
    _configRepo = ref.watch(configRepoProvider);
    _taskRepo = ref.watch(taskRepoProvider);
    _storage = ref.watch(localStorageProvider);
    return const DashboardState();
  }

  Future<void> load() async {
    state = state.copyWith(loading: true);
    try {
      final results = await Future.wait([
        _configRepo.getConfig(forceRefresh: true),
        _taskRepo.listTasks(size: 5),
      ]);
      state = state.copyWith(
        config: results[0] as AppConfig,
        recentTasks: results[1] as List<Task>,
      );
    } catch (_) {}
    try {
      final cw = await _storage.getList('custom_workflows');
      state = state.copyWith(
        customWorkflows: cw.map(TaskRecipe.fromJson).toList(),
      );
    } catch (_) {}
    state = state.copyWith(loading: false);
  }

  Future<void> saveCustomWorkflow(TaskRecipe workflow) async {
    final updated = [
      ...state.customWorkflows.where((item) => item.name != workflow.name),
      workflow,
    ];
    await _storage.setList(
      'custom_workflows',
      updated.map((item) => item.toStorageJson()).toList(),
    );
    state = state.copyWith(customWorkflows: updated);
  }
}

final dashboardProvider = NotifierProvider<DashboardNotifier, DashboardState>(
  DashboardNotifier.new,
);
