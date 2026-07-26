import 'package:file_picker/file_picker.dart';
import 'package:flutter/cupertino.dart';
import 'package:provider/provider.dart';

import '../services/api_client.dart';

class CreateTaskPage extends StatefulWidget {
  const CreateTaskPage({super.key});

  @override
  State<CreateTaskPage> createState() => _CreateTaskPageState();
}

class _CreateTaskPageState extends State<CreateTaskPage> {
  final _pathController = TextEditingController();
  final _outputController = TextEditingController();
  final _langController = TextEditingController(text: 'zh');
  int _mode = 3;
  bool _submitting = false;

  @override
  void dispose() {
    _pathController.dispose();
    _outputController.dispose();
    _langController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      height: MediaQuery.of(context).size.height * 0.65,
      decoration: const BoxDecoration(
        color: CupertinoColors.systemGroupedBackground,
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      child: Column(
        children: [
          _buildHeader(),
          Expanded(child: _buildForm()),
        ],
      ),
    );
  }

  Widget _buildHeader() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: const BoxDecoration(
        border: Border(bottom: BorderSide(color: Color(0xFFE5E5EA), width: 0.5)),
      ),
      child: Row(
        children: [
          CupertinoButton(
            padding: EdgeInsets.zero,
            onPressed: () => Navigator.pop(context),
            child: const Text('取消', style: TextStyle(fontSize: 15)),
          ),
          const Expanded(
            child: Text(
              '新建任务',
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600, color: Color(0xFF1D1D1F)),
            ),
          ),
          CupertinoButton(
            padding: EdgeInsets.zero,
            onPressed: _submitting ? null : _submit,
            child: Text(
              '创建',
              style: TextStyle(
                fontSize: 15,
                fontWeight: FontWeight.w600,
                color: _submitting ? const Color(0xFFC7C7CC) : CupertinoColors.systemBlue,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildForm() {
    return ListView(
      padding: const EdgeInsets.all(20),
      children: [
        _sectionLabel('视频文件'),
        Row(
          children: [
            Expanded(
              child: CupertinoTextField(
                controller: _pathController,
                placeholder: '选择或输入视频文件路径',
                padding: const EdgeInsets.all(12),
                decoration: _fieldDecoration(),
              ),
            ),
            const SizedBox(width: 8),
            CupertinoButton(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              color: const Color(0xFFE5E5EA),
              borderRadius: BorderRadius.circular(8),
              onPressed: _pickFile,
              child: const Text('浏览', style: TextStyle(color: Color(0xFF3A3A3C), fontSize: 13)),
            ),
          ],
        ),
        const SizedBox(height: 20),
        _sectionLabel('处理模式'),
        CupertinoSlidingSegmentedControl<int>(
          groupValue: _mode,
          children: const {
            1: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('字幕', style: TextStyle(fontSize: 13))),
            2: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('翻译', style: TextStyle(fontSize: 13))),
            3: Padding(padding: EdgeInsets.symmetric(horizontal: 12), child: Text('配音', style: TextStyle(fontSize: 13))),
          },
          onValueChanged: (v) => setState(() => _mode = v!),
        ),
        const SizedBox(height: 20),
        _sectionLabel('目标语言'),
        CupertinoTextField(
          controller: _langController,
          placeholder: 'zh (默认中文)',
          padding: const EdgeInsets.all(12),
          decoration: _fieldDecoration(),
        ),
        const SizedBox(height: 20),
        _sectionLabel('输出目录（留空自动生成）'),
        CupertinoTextField(
          controller: _outputController,
          placeholder: '自动生成到视频同级目录',
          padding: const EdgeInsets.all(12),
          decoration: _fieldDecoration(),
        ),
      ],
    );
  }

  BoxDecoration _fieldDecoration() {
    return BoxDecoration(
      color: CupertinoColors.white,
      borderRadius: BorderRadius.circular(8),
      border: Border.all(color: const Color(0xFFE5E5EA)),
    );
  }

  Widget _sectionLabel(String text) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Text(
        text,
        style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500, color: Color(0xFF8E8E93)),
      ),
    );
  }

  Future<void> _pickFile() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.video,
      dialogTitle: '选择视频文件',
    );
    if (result != null && result.files.single.path != null) {
      _pathController.text = result.files.single.path!;
    }
  }

  Future<void> _submit() async {
    if (_pathController.text.trim().isEmpty) return;
    setState(() => _submitting = true);
    try {
      await context.read<TaskListNotifier>().createTask(
            inputPath: _pathController.text.trim(),
            mode: _mode,
            outputDir: _outputController.text.trim(),
            targetLang: _langController.text.trim(),
          );
      if (mounted) Navigator.pop(context);
    } on ApiException catch (e) {
      if (mounted) {
        _showError(e.message);
      }
    } catch (e) {
      if (mounted) {
        _showError('创建失败: $e');
      }
    }
    if (mounted) setState(() => _submitting = false);
  }

  void _showError(String msg) {
    showCupertinoDialog(
      context: context,
      builder: (_) => CupertinoAlertDialog(
        title: const Text('错误'),
        content: Text(msg),
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
