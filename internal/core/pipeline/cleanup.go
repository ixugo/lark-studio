package pipeline

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ixugo/vdub/internal/taskfile"
)

// cleanIntermediate 删除输出目录中的中间产物，保留最终交付件
// 返回删除的文件/目录数量
//
// 保留: *.mp4, src.srt, trans.srt, task.log
// 删除: raw.mp3, trans.txt, concat_list.txt, dub.mp3, silence_*.wav, audio_segs/
func cleanIntermediate(outputDir string) int {
	keep := map[string]bool{
		"src.srt":          true,
		"trans.srt":        true,
		"task.log":         true,
		"source_meta.json": true,
	}

	if meta, err := taskfile.Read(outputDir); err == nil {
		keep[meta.StagedName] = true
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		slog.Debug("cleanup: read dir failed", "dir", outputDir, "err", err)
		return 0
	}

	removed := 0
	for _, e := range entries {
		name := e.Name()

		if keep[name] || strings.HasPrefix(name, "src.") {
			continue
		}
		if strings.HasSuffix(name, ".mp4") {
			continue
		}

		fullPath := filepath.Join(outputDir, name)
		if e.IsDir() {
			if err := os.RemoveAll(fullPath); err != nil {
				slog.Debug("cleanup: remove dir failed", "path", fullPath, "err", err)
				continue
			}
		} else {
			if err := os.Remove(fullPath); err != nil {
				slog.Debug("cleanup: remove file failed", "path", fullPath, "err", err)
				continue
			}
		}
		removed++
	}
	return removed
}

// CleanStepAndSubsequent 重跑指定节点时，清除该节点及其后续所有节点的产物文件，保留前置节点的产物
func CleanStepAndSubsequent(outputDir string, fromStep string) int {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return 0
	}

	// 永远保护的任务元数据与日志
	protected := map[string]bool{
		"source_meta.json": true,
		"task.log":         true,
	}

	if meta, err := taskfile.Read(outputDir); err == nil {
		protected[meta.StagedName] = true
	}
	stepOrder := map[string]int{
		StepWhisper:   1,
		StepSplit:     2,
		StepTranslate: 3,
		StepTTS:       4,
		StepMerge:     5,
		StepBurn:      6,
	}

	fromRank := stepOrder[fromStep]
	if fromRank <= 0 {
		fromRank = 1
	}

	removed := 0
	for _, e := range entries {
		name := e.Name()
		if protected[name] {
			continue
		}

		shouldDelete := false

		// 1. whisper 产物: raw.mp3, src.srt
		if fromRank <= stepOrder[StepWhisper] {
			if name == "raw.mp3" || name == "src.srt" {
				shouldDelete = true
			}
		}

		// 2. translate 产物: trans.txt, trans.srt
		if fromRank <= stepOrder[StepTranslate] {
			if name == "trans.txt" || name == "trans.srt" {
				shouldDelete = true
			}
		}

		// 3. tts 产物: audio_segs 目录
		if fromRank <= stepOrder[StepTTS] {
			if name == "audio_segs" {
				shouldDelete = true
			}
		}

		// 4. merge 产物: dub.mp3, concat_list.txt, silence_*.wav, intermediate 目录
		if fromRank <= stepOrder[StepMerge] {
			if name == "dub.mp3" || name == "concat_list.txt" || name == "intermediate" || strings.HasPrefix(name, "silence_") {
				shouldDelete = true
			}
		}

		// 5. burn 产物：统一成片 output.mp4，同时清理旧版本成片。
		if fromRank <= stepOrder[StepBurn] {
			if name == "output.mp4" || strings.HasSuffix(name, ".sub.mp4") || strings.HasSuffix(name, ".trans.mp4") || strings.HasSuffix(name, ".final.mp4") {
				shouldDelete = true
			}
		}

		if shouldDelete {
			fullPath := filepath.Join(outputDir, name)
			if e.IsDir() {
				_ = os.RemoveAll(fullPath)
			} else {
				_ = os.Remove(fullPath)
			}
			removed++
		}
	}
	if result, err := taskfile.ResultVideoPath(outputDir); err == nil && filepath.Dir(result) != filepath.Clean(outputDir) {
		if err := os.Remove(result); err == nil {
			removed++
		} else if !os.IsNotExist(err) {
			slog.Warn("重跑清理成片失败", "path", result, "err", err)
		}
	}
	return removed
}
