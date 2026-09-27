package task

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixugo/goddd/pkg/orm"
)

// removeTaskFiles 只清理本任务可确定归属的产物，避免自选目录中的无关文件被删除。
func (c Core) removeTaskFiles(ctx context.Context, item *Task) error {
	if item.OutputDir == "" {
		return nil
	}
	slog.DebugContext(ctx, "清理任务产物", "task_id", item.ID, "output_dir", item.OutputDir)
	count, err := c.store.Task().Count(ctx, orm.Where("output_dir=? AND id<>?", item.OutputDir, item.ID))
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("输出目录被其他任务共用，无法安全删除产物")
	}
	entries, err := os.ReadDir(item.OutputDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取任务产物目录失败: %w", err)
	}
	paths, err := taskArtifactPaths(item, entries)
	if err != nil {
		return err
	}
	return deleteTaskArtifactPaths(item.OutputDir, paths)
}

// taskArtifactPaths 以 O(n) 扫描产物，并在任何删除之前检查原文件保护条件。
func taskArtifactPaths(item *Task, entries []os.DirEntry) ([]string, error) {
	input, staged, err := taskOriginalSource(item)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.Name() == "source_meta.json" {
			continue
		}
		if entry.Name() != staged && !isTaskArtifact(item.InputPath, entry.Name()) {
			continue
		}
		path := filepath.Join(item.OutputDir, entry.Name())
		if err := protectTaskInput(input, path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// deleteTaskArtifactPaths 最后删除源文件记录，确保中途失败仍能重试并识别工作副本。
func deleteTaskArtifactPaths(outputDir string, paths []string) error {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("删除任务产物 %s 失败: %w", filepath.Base(path), err)
		}
	}
	if err := os.Remove(filepath.Join(outputDir, "source_meta.json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	remaining, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return os.Remove(outputDir)
	}
	return nil
}

// taskOriginalSource 识别应用生成的工作副本，删除副本时仍保护用户原文件。
func taskOriginalSource(item *Task) (string, string, error) {
	data, err := os.ReadFile(filepath.Join(item.OutputDir, "source_meta.json"))
	if os.IsNotExist(err) {
		return item.InputPath, "", nil
	}
	if err != nil {
		return "", "", err
	}
	var meta struct {
		OriginalPath string `json:"original_path"`
		StagedPath   string `json:"staged_path"`
		StagedName   string `json:"staged_name"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", "", fmt.Errorf("读取源文件记录失败: %w", err)
	}
	if meta.OriginalPath == "" || meta.StagedName == "" || filepath.Base(meta.StagedName) != meta.StagedName {
		return "", "", fmt.Errorf("源文件记录不完整，无法安全删除工作副本")
	}
	staged := filepath.Clean(filepath.Join(item.OutputDir, meta.StagedName))
	if staged != filepath.Clean(meta.StagedPath) || staged != filepath.Clean(item.InputPath) || staged == filepath.Clean(meta.OriginalPath) {
		return "", "", fmt.Errorf("源文件记录与任务不一致，无法安全删除工作副本")
	}
	return meta.OriginalPath, meta.StagedName, nil
}

// isTaskArtifact 根据流水线的固定命名识别产物，成片必须匹配本任务的源文件名。
func isTaskArtifact(input, name string) bool {
	switch name {
	case "src.srt", "trans.srt", "trans.txt", "raw.mp3", "dub.mp3", "concat_list.txt",
		"audio_segs", "lipsync.mp4", "task.log", "source_meta.json":
		return true
	}
	if strings.HasPrefix(name, "silence_") && strings.HasSuffix(name, ".wav") {
		return true
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	return name == base+".final.mp4" || name == base+".sub.mp4" || name == base+".trans.mp4"
}

// protectTaskInput 检查真实路径，拒绝删除与原素材相同或包含原素材的产物位置。
func protectTaskInput(input, artifact string) error {
	if input == "" {
		return nil
	}
	source, err := filepath.EvalSymlinks(input)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	target, err := filepath.EvalSymlinks(artifact)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(target, source)
	if err != nil {
		return err
	}
	if relative == "." || filepath.IsLocal(relative) {
		return fmt.Errorf("产物位置包含原始素材，已停止删除: %s", artifact)
	}
	return nil
}
