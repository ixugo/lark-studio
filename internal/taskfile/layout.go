// Package taskfile 管理源文件记录及任务工作目录，不依赖任务领域或流水线。
package taskfile

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"uuid"
)

type Metadata struct {
	LayoutVersion int       `json:"layout_version,omitempty"`
	OriginalName  string    `json:"original_name"`
	OriginalPath  string    `json:"original_path"`
	StagedName    string    `json:"staged_name"`
	StagedPath    string    `json:"staged_path"`
	ResultName    string    `json:"result_name,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func Read(workDir string) (Metadata, error) {
	var meta Metadata
	data, err := os.ReadFile(filepath.Join(workDir, "source_meta.json"))
	if err != nil {
		return meta, err
	}
	err = json.Unmarshal(data, &meta)
	return meta, err
}

// NewRoot 使用原文件名建立单源目录，多源使用批次 UUID；重复提交不会覆盖已有任务。
func NewRoot(base string, inputs []string) (string, error) {
	if err := os.MkdirAll(base, 0755); err != nil {
		return "", err
	}
	name := "batch_" + uuid.NewV4().String()
	if len(inputs) == 1 {
		name = strings.TrimSuffix(filepath.Base(inputs[0]), filepath.Ext(inputs[0])) + "_lark_studio"
	}
	for {
		path := filepath.Join(base, name)
		err := os.Mkdir(path, 0755)
		if err == nil {
			return path, nil
		}
		if !os.IsExist(err) {
			return "", err
		}
		name += "_" + uuid.NewV4().String()
	}
}

// ResultVideoPath 只接受本工作目录的成片名称，禁止记录指定任意外部路径。
func ResultVideoPath(workDir string) (string, error) {
	meta, err := Read(workDir)
	if os.IsNotExist(err) {
		return filepath.Join(workDir, "output.mp4"), nil
	}
	if err != nil {
		return "", err
	}
	if meta.LayoutVersion == 0 {
		return filepath.Join(workDir, "output.mp4"), nil
	}
	if meta.LayoutVersion != 1 {
		return "", fmt.Errorf("未知任务目录版本")
	}
	id := filepath.Base(workDir)
	parsed, err := uuid.Parse(id)
	if err != nil || parsed[6]>>4 != 4 || parsed.String() != id {
		return "", fmt.Errorf("工作目录不是 UUIDv4")
	}
	if meta.ResultName != "output.mp4" && meta.ResultName != id+".mp4" {
		return "", fmt.Errorf("成片名称与工作目录不一致")
	}
	if meta.StagedName != id+filepath.Ext(meta.StagedName) || filepath.Base(meta.StagedName) != meta.StagedName || filepath.Clean(meta.StagedPath) != filepath.Join(filepath.Clean(workDir), meta.StagedName) {
		return "", fmt.Errorf("源文件记录与工作目录不一致")
	}
	return filepath.Join(filepath.Dir(workDir), meta.ResultName), nil
}
