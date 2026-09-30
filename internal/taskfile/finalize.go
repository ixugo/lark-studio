package taskfile

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FinalizeResult 先原子创建不覆盖的硬链接，再记录最终路径并移除渲染名。
// 同目录无需复制视频；碰撞扫描为 O(k)，k 为已有同名后缀数量。
func FinalizeResult(workDir string) error {
	meta, err := Read(workDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if meta.LayoutVersion != 1 {
		return nil
	}
	rendered, err := RenderVideoPath(workDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(rendered); os.IsNotExist(err) && meta.FinalName != "" {
		path, err := ResultVideoPath(workDir)
		if err != nil {
			return err
		}
		_, err = os.Stat(path)
		return err
	} else if err != nil {
		return err
	}
	if meta.OriginalName == "" || filepath.Base(meta.OriginalName) != meta.OriginalName {
		return fmt.Errorf("原始文件名无效")
	}
	return publishResult(workDir, meta, rendered)
}

// publishResult 原子争用名称，元数据落盘失败时保留 UUID 成片供重试。
func publishResult(workDir string, meta Metadata, rendered string) error {
	base := strings.TrimSuffix(meta.OriginalName, filepath.Ext(meta.OriginalName))
	for index := 0; ; index++ {
		name := base + ".mp4"
		if index > 0 {
			name = fmt.Sprintf("%s_%d.mp4", base, index)
		}
		target := filepath.Join(filepath.Dir(rendered), name)
		if err := os.Link(rendered, target); os.IsExist(err) {
			continue
		} else if err != nil {
			return err
		}
		meta.FinalName = name
		if err := writeMetadata(workDir, meta); err != nil {
			if cleanupErr := os.Remove(target); cleanupErr != nil {
				return fmt.Errorf("保存成片记录失败: %v；撤销名称失败: %w", err, cleanupErr)
			}
			return err
		}
		return os.Remove(rendered)
	}
}

func writeMetadata(workDir string, meta Metadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(workDir, "source_meta-*.json")
	if err != nil {
		return err
	}
	name := temp.Name()
	if _, err := temp.Write(data); err != nil {
		return closeMetadataTemp(temp, err)
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, filepath.Join(workDir, "source_meta.json")); err != nil {
		if cleanupErr := os.Remove(name); cleanupErr != nil {
			return fmt.Errorf("更新记录失败: %v；清理失败: %w", err, cleanupErr)
		}
		return err
	}
	return nil
}

func closeMetadataTemp(file *os.File, writeErr error) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("写记录失败: %v；关闭失败: %w", writeErr, err)
	}
	if err := os.Remove(file.Name()); err != nil {
		return fmt.Errorf("写记录失败: %v；清理失败: %w", writeErr, err)
	}
	return writeErr
}
