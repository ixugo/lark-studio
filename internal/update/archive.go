package update

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	maxExpandedBytes = 2 << 30
	maxArchiveFiles  = 4096
)

// Windows 的路径规则比 Unix 更严格；在任何系统上都按 Windows 规则检查 zip。
func archivePath(name string) (string, error) {
	if !utf8.ValidString(name) || len(name) > 1024 || strings.ContainsAny(name, "\\:\x00") {
		return "", errors.New("安装包路径不安全")
	}
	clean := strings.TrimSuffix(name, "/")
	if clean == "" || strings.HasPrefix(clean, "/") {
		return "", errors.New("安装包包含绝对路径")
	}
	parts := strings.Split(clean, "/")
	if parts[0] != "lark-studio" {
		return "", errors.New("安装包必须包含 lark-studio 应用目录")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", errors.New("安装包包含不安全路径段")
		}
		for _, r := range part {
			if r < 32 {
				return "", errors.New("安装包路径包含控制字符")
			}
		}
		base, _, _ := strings.Cut(strings.ToUpper(part), ".")
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
			return "", errors.New("安装包路径包含 Windows 保留名")
		}
	}
	return filepath.Join(parts...), nil
}

func extractZip(ctx context.Context, archive, destination string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("打开安装包: %w", err)
	}
	defer r.Close()
	if len(r.File) > maxArchiveFiles {
		return errors.New("安装包文件数量超过限制")
	}
	var expanded uint64
	seen := make(map[string]bool, len(r.File))
	for _, entry := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := archivePath(entry.Name)
		if err != nil {
			return err
		}
		key := strings.ToLower(rel)
		if seen[key] {
			return errors.New("安装包包含重复文件路径")
		}
		seen[key] = true
		if entry.Mode()&os.ModeType != 0 && !entry.Mode().IsDir() {
			return errors.New("安装包包含链接或特殊文件")
		}
		expanded += entry.UncompressedSize64
		if expanded > maxExpandedBytes {
			return errors.New("安装包解压大小超过限制")
		}
		if err := extractEntry(entry, filepath.Join(destination, rel)); err != nil {
			return err
		}
	}
	return nil
}

func extractEntry(entry *zip.File, path string) error {
	if entry.FileInfo().IsDir() {
		return os.MkdirAll(path, 0700)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	input, err := entry.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(output, io.LimitReader(input, int64(entry.UncompressedSize64)+1))
	err = errors.Join(copyErr, output.Close())
	if err != nil {
		return err
	}
	if n != int64(entry.UncompressedSize64) {
		return errors.New("安装包解压文件大小不符")
	}
	return nil
}
