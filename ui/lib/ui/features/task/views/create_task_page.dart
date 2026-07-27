import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:hooks_riverpod/hooks_riverpod.dart';

import '../../../../data/services/api_client.dart';
import '../view_models/task_list_view_model.dart';

class CreateTaskPage extends HookConsumerWidget {
  const CreateTaskPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final pathController = useTextEditingController();
    final outputController = useTextEditingController();
    final langController = useTextEditingController(text: 'zh');
    final mode = useState(3);
    final submitting = useState(false);

    Future<void> pickFile() async {
      final result = await FilePicker.platform.pickFiles(type: FileType.video, dialogTitle: '选择视频文件');
      if (result != null && result.files.single.path != null) {
        pathController.text = result.files.single.path!;
      }
    }

    Future<void> submit() async {
      if (pathController.text.trim().isEmpty) return;
      submitting.value = true;
      try {
        await ref.read(taskListProvider.notifier).createTask(
              inputPath: pathController.text.trim(),
              mode: mode.value,
              outputDir: outputController.text.trim(),
              targetLang: langController.text.trim(),
            );
        if (context.mounted) Navigator.pop(context);
      } on ApiException catch (e) {
        if (context.mounted) _showError(context, e.message);
      } catch (e) {
        if (context.mounted) _showError(context, '创建失败: $e');
      }
      submitting.value = false;
    }

    return Container(
      height: MediaQuery.of(context).size.height * 0.65,
      decoration: const BoxDecoration(
        color: CupertinoColors.systemGroupedBackground,
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: Column(children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5))),
          child: Row(children: [
            CupertinoButton(padding: EdgeInsets.zero, onPressed: () => Navigator.pop(context), child: const Text('取消', style: TextStyle(fontSize: 15))),
            const Expanded(child: Text('新建任务', textAlign: TextAlign.center, style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F)))),
            CupertinoButton(padding: EdgeInsets.zero, onPressed: submitting.value ? null : submit,
                child: Text('创建', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: submitting.value ? const Color(0xFFC7C7CC) : CupertinoColors.systemBlue))),
          ]),
        ),
        Expanded(child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            _sectionLabel('视频文件'),
            Row(children: [
              Expanded(child: CupertinoTextField(controller: pathController, placeholder: '选择或输入视频文件路径', padding: const EdgeInsets.all(12), decoration: _fieldDecoration())),
              const SizedBox(width: 8),
              CupertinoButton(padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8), color: const Color(0xFFE5E5EA), borderRadius: BorderRadius.circular(8), onPressed: pickFile, child: const Text('浏览', style: TextStyle(color: Color(0xFF3A3A3C), fontSize: 13))),
            ]),
            const SizedBox(height: 20),
            _sectionLabel('处理模式'),
            CupertinoSlidingSegmentedControl<int>(groupValue: mode.value, children: const {
              1: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('字幕', style: TextStyle(fontSize: 13))),
              2: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('翻译', style: TextStyle(fontSize: 13))),
              3: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('配音', style: TextStyle(fontSize: 13))),
            }, onValueChanged: (v) => mode.value = v!),
            const SizedBox(height: 20),
            _sectionLabel('目标语言'),
            CupertinoTextField(controller: langController, placeholder: 'zh (默认中文)', padding: const EdgeInsets.all(12), decoration: _fieldDecoration()),
            const SizedBox(height: 20),
            _sectionLabel('输出目录（留空自动生成）'),
            CupertinoTextField(controller: outputController, placeholder: '自动生成到视频同级目录', padding: const EdgeInsets.all(12), decoration: _fieldDecoration()),
          ],
        )),
      ]),
    );
  }

  BoxDecoration _fieldDecoration() => BoxDecoration(color: CupertinoColors.white, borderRadius: BorderRadius.circular(8), border: Border.all(color: const Color(0xFFE5E5EA)));

  Widget _sectionLabel(String text) => Padding(padding: const EdgeInsets.only(bottom: 8), child: Text(text, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93))));

  void _showError(BuildContext context, String msg) {
    showCupertinoDialog(context: context, builder: (_) => CupertinoAlertDialog(title: const Text('错误'), content: Text(msg), actions: [CupertinoDialogAction(child: const Text('确定'), onPressed: () => Navigator.pop(context))]));
  }
}
