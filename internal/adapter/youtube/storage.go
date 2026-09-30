package youtube

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Storage struct {
	Directory string `json:"directory"`
	Bytes     int64  `json:"bytes"`
	Files     int    `json:"files"`
}

func storageFiles(dir string) ([]os.DirEntry, error) {
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("下载目录必须是普通文件夹")
	}
	return os.ReadDir(dir)
}

func (m *Manager) Storage(dir string) (Storage, error) {
	result := Storage{Directory: dir}
	files, err := storageFiles(dir)
	if err != nil {
		return result, err
	}
	for _, entry := range files {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".mp4") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return result, err
		}
		result.Bytes += info.Size()
		result.Files++
	}
	return result, nil
}

func (m *Manager) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if busy(m.status.Phase) {
		return errors.New("请等待当前下载操作完成")
	}
	m.resetLocked()
	return nil
}

func (m *Manager) resetLocked() {
	m.status = Status{Phase: "idle"}
	m.info = nil
	m.session = Session{}
}

func (m *Manager) ClearStorage(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if busy(m.status.Phase) {
		return errors.New("下载进行中，暂不能清理视频")
	}
	files, err := storageFiles(dir)
	if err != nil {
		return err
	}
	for _, entry := range files {
		if !entry.Type().IsRegular() || !strings.EqualFold(filepath.Ext(entry.Name()), ".mp4") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	m.resetLocked()
	return nil
}
