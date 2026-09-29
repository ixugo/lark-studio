package task

import (
	"path/filepath"

	"github.com/ixugo/vdub/internal/taskfile"
)

// enrichTaskFiles 从已有源文件记录还原展示名，不增加数据库字段。
func enrichTaskFiles(item *Task) {
	item.OriginalName = filepath.Base(item.InputPath)
	item.BatchDir = item.OutputDir
	meta, err := taskfile.Read(item.OutputDir)
	if err != nil {
		return
	}
	if meta.OriginalName != "" {
		item.OriginalName = meta.OriginalName
	}
	path, err := taskfile.ResultVideoPath(item.OutputDir)
	if err != nil {
		return
	}
	item.ResultPath = path
	if meta.LayoutVersion == 1 {
		item.BatchDir = filepath.Dir(item.OutputDir)
	}
}
