package pipeline

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// cleanIntermediate 删除输出目录中的中间产物，保留最终交付件
// 返回删除的文件/目录数量
//
// 保留: *.mp4, src.srt, trans.srt, task.log
// 删除: raw.mp3, trans.txt, concat_list.txt, dub.mp3, silence_*.wav, audio_segs/
func cleanIntermediate(outputDir string) int {
	keep := map[string]bool{
		"src.srt":   true,
		"trans.srt": true,
		"task.log":  true,
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		slog.Debug("cleanup: read dir failed", "dir", outputDir, "err", err)
		return 0
	}

	removed := 0
	for _, e := range entries {
		name := e.Name()

		if keep[name] {
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
