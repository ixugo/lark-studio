package api

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	whisperadapter "github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/pkg/web"
	"github.com/ixugo/vdub/pkg/ws"
)

const hfBaseURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"

type whisperModelInfo struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Desc string `json:"desc"`
}

var whisperModels = []whisperModelInfo{
	{"tiny", "75 MiB", "最小最快，适合快速测试"},
	{"tiny-q5_1", "32.2 MiB", "最小量化版，适合快速试用"},
	{"tiny-q8_0", "43.5 MiB", "最小高精度量化版"},
	{"tiny.en", "77.7 MiB", "英文专用，最小模型"},
	{"tiny.en-q5_1", "32.2 MiB", "英文专用最小量化版"},
	{"tiny.en-q8_0", "43.6 MiB", "英文专用最小高精度量化版"},
	{"base", "148 MiB", "速度与质量均衡，轻量推荐"},
	{"base-q5_1", "59.7 MiB", "基础量化版，内存占用低"},
	{"base-q8_0", "81.8 MiB", "基础高精度量化版"},
	{"base.en", "148 MiB", "英文专用基础模型"},
	{"base.en-q5_1", "59.7 MiB", "英文专用基础量化版"},
	{"base.en-q8_0", "81.8 MiB", "英文专用基础高精度量化版"},
	{"small", "488 MiB", "速度与质量平衡，适合常规字幕"},
	{"small-q5_1", "190 MiB", "小模型量化版"},
	{"small-q8_0", "264 MiB", "小模型高精度量化版"},
	{"small.en", "488 MiB", "英文专用小模型"},
	{"small.en-q5_1", "190 MiB", "英文专用小模型量化版"},
	{"small.en-q8_0", "264 MiB", "英文专用小模型高精度量化版"},
	{"medium", "1.53 GiB", "中文效果好，适合多数正式任务"},
	{"medium-q5_0", "539 MiB", "中型量化版，兼顾精度与内存"},
	{"medium-q8_0", "823 MiB", "中型高精度量化版"},
	{"medium.en", "1.53 GiB", "英文专用中型模型"},
	{"medium.en-q5_0", "539 MiB", "英文专用中型量化版"},
	{"medium.en-q8_0", "823 MiB", "英文专用中型高精度量化版"},
	{"large-v3-turbo", "1.62 GiB", "精度接近旗舰，速度更佳"},
	{"large-v3-turbo-q5_0", "574 MiB", "生产推荐，旗舰量化平衡版"},
	{"large-v3-turbo-q8_0", "874 MiB", "旗舰 Turbo 高精度量化版"},
	{"large-v3", "3.1 GiB", "当前旗舰，最高精度"},
	{"large-v3-q5_0", "1.08 GiB", "当前旗舰量化版"},
	{"large-v2", "3.09 GiB", "上一代旗舰模型"},
	{"large-v2-q5_0", "1.08 GiB", "上一代旗舰量化版"},
	{"large-v2-q8_0", "1.66 GiB", "上一代旗舰高精度量化版"},
	{"large-v1", "3.09 GiB", "第一代旗舰模型"},
}

type modelDownloadStatus struct {
	mu       sync.Mutex
	active   map[string]bool
	progress map[string]int
}

var dlStatus = &modelDownloadStatus{
	active:   make(map[string]bool),
	progress: make(map[string]int),
}

var (
	runtimeInstallMu  sync.Mutex
	runtimeInstalling bool
)

func (s *modelDownloadStatus) start(name string) {
	s.mu.Lock()
	s.active[name] = true
	s.progress[name] = 0
	s.mu.Unlock()
}

func (s *modelDownloadStatus) finish(name string) {
	s.mu.Lock()
	delete(s.active, name)
	delete(s.progress, name)
	s.mu.Unlock()
}

func (s *modelDownloadStatus) setProgress(name string, p int) {
	s.mu.Lock()
	s.progress[name] = p
	s.mu.Unlock()
}

func (s *modelDownloadStatus) isActive(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[name]
}

func (s *modelDownloadStatus) getProgress(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress[name]
}

func modelsDir() string {
	return filepath.Join(conf.DataDir(), "models")
}

func modelPath(name string) string {
	return filepath.Join(modelsDir(), fmt.Sprintf("ggml-%s.bin", name))
}

// RegisterModel 注册 Whisper 运行时与模型管理路由。
func RegisterModel(mux *http.ServeMux, hub ws.Huber, cfg *conf.Bootstrap) {
	mux.HandleFunc("GET /models", web.WrapH(listModels))
	mux.HandleFunc("POST /models/download", web.WrapH(func(_ *http.Request, in *modelDownloadInput) (any, error) {
		return startDownload(in.Name, hub)
	}))
	mux.HandleFunc("GET /models/runtime", web.WrapH(func(_ *http.Request, _ *struct{}) (any, error) {
		return inspectWhisperRuntime(cfg), nil
	}))
	mux.HandleFunc("POST /models/runtime/install", web.WrapH(func(_ *http.Request, _ *struct{}) (any, error) {
		return startRuntimeInstall(hub, cfg)
	}))
}

type whisperRuntimeOutput struct {
	whisperadapter.RuntimeInfo
	Installing bool `json:"installing"`
}

func inspectWhisperRuntime(cfg *conf.Bootstrap) whisperRuntimeOutput {
	runtimeInstallMu.Lock()
	installing := runtimeInstalling
	runtimeInstallMu.Unlock()
	return whisperRuntimeOutput{
		RuntimeInfo: whisperadapter.InspectRuntime(cfg.Pipeline.WhisperBin),
		Installing:  installing,
	}
}

func startRuntimeInstall(hub ws.Huber, cfg *conf.Bootstrap) (any, error) {
	if info := whisperadapter.InspectRuntime(cfg.Pipeline.WhisperBin); info.Installed {
		return map[string]string{"status": "already_installed"}, nil
	}

	runtimeInstallMu.Lock()
	if runtimeInstalling {
		runtimeInstallMu.Unlock()
		return map[string]string{"status": "installing"}, nil
	}
	runtimeInstalling = true
	runtimeInstallMu.Unlock()

	go func() {
		defer func() {
			runtimeInstallMu.Lock()
			runtimeInstalling = false
			runtimeInstallMu.Unlock()
		}()

		err := whisperadapter.InstallRuntime(context.Background(), whisperadapter.RuntimeInstallOptions{
			ReleaseTag: cfg.Runtime.BuildVersion,
		}, func(line string) {
			if hub != nil {
				hub.Broadcast(ws.NewMessage("whisper_runtime_log", map[string]any{
					"message": line,
				}))
			}
		})
		if err != nil {
			slog.Error("whisper runtime install failed", "err", err)
			if hub != nil {
				hub.Broadcast(ws.NewMessage("whisper_runtime_failed", map[string]any{
					"error": err.Error(),
				}))
			}
			return
		}
		if hub != nil {
			hub.Broadcast(ws.NewMessage("whisper_runtime_done", inspectWhisperRuntime(cfg)))
		}
	}()
	return map[string]string{"status": "started"}, nil
}

type modelListOutput struct {
	Name        string `json:"name"`
	Size        string `json:"size"`
	Desc        string `json:"desc"`
	Downloaded  bool   `json:"downloaded"`
	Path        string `json:"path,omitempty"`
	Downloading bool   `json:"downloading"`
	Progress    int    `json:"progress"`
}

func listModels(_ *http.Request, _ *struct{}) ([]modelListOutput, error) {
	var out []modelListOutput
	for _, m := range whisperModels {
		p := modelPath(m.Name)
		_, err := os.Stat(p)
		downloaded := err == nil
		item := modelListOutput{
			Name:        m.Name,
			Size:        m.Size,
			Desc:        m.Desc,
			Downloaded:  downloaded,
			Downloading: dlStatus.isActive(m.Name),
			Progress:    dlStatus.getProgress(m.Name),
		}
		if downloaded {
			item.Path = p
		}
		out = append(out, item)
	}
	return out, nil
}

type modelDownloadInput struct {
	Name string `json:"name"`
}

func startDownload(name string, hub ws.Huber) (any, error) {
	valid := false
	for _, m := range whisperModels {
		if m.Name == name {
			valid = true
			break
		}
	}
	if !valid {
		return nil, fmt.Errorf("unknown model: %s", name)
	}

	p := modelPath(name)
	if _, err := os.Stat(p); err == nil {
		return map[string]string{"path": p, "status": "already_exists"}, nil
	}

	if dlStatus.isActive(name) {
		return map[string]string{"status": "downloading"}, nil
	}

	dlStatus.start(name)
	go func() {
		defer dlStatus.finish(name)
		if err := downloadModel(name, hub); err != nil {
			slog.Error("model download failed", "model", name, "err", err)
			if hub != nil {
				hub.Broadcast(ws.NewMessage("model_download_failed", map[string]any{
					"model": name, "error": err.Error(),
				}))
			}
		}
	}()

	return map[string]string{"status": "started"}, nil
}

func downloadModel(name string, hub ws.Huber) error {
	url := fmt.Sprintf("%s/ggml-%s.bin", hfBaseURL, name)
	dest := modelPath(name)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("创建模型目录失败: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	tmpFile := dest + ".tmp"
	f, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 64*1024)
	lastPct := -1

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				f.Close()
				os.Remove(tmpFile)
				return fmt.Errorf("写入失败: %w", wErr)
			}
			written += int64(n)
			if total > 0 {
				pct := int(written * 100 / total)
				if pct != lastPct {
					lastPct = pct
					dlStatus.setProgress(name, pct)
					if hub != nil {
						hub.Broadcast(ws.NewMessage("model_download_progress", map[string]any{
							"model": name, "progress": pct,
						}))
					}
				}
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			f.Close()
			os.Remove(tmpFile)
			return fmt.Errorf("读取失败: %w", readErr)
		}
	}
	f.Close()

	if err := os.Rename(tmpFile, dest); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("重命名失败: %w", err)
	}

	slog.Info("model downloaded", "model", name, "path", dest, "size", written)
	if hub != nil {
		hub.Broadcast(ws.NewMessage("model_download_done", map[string]any{
			"model": name, "path": dest,
		}))
	}
	return nil
}
