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

	"github.com/gin-gonic/gin"
	whisperadapter "github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/ixugo/vdub/pkg/web"
	"github.com/ixugo/vdub/pkg/ws"
)

const hfBaseURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"

// whisperModelInfo 模型元数据，硬编码自 HuggingFace 目录
type whisperModelInfo struct {
	Name string `json:"name"`
	Size string `json:"size"`
	Desc string `json:"desc"`
}

var whisperModels = []whisperModelInfo{
	{"tiny", "75 MiB", "最小最快，适合快速测试"},
	{"base", "142 MiB", "默认推荐，速度与质量均衡"},
	{"small", "466 MiB", "精度更高，适合正式字幕"},
	{"medium", "1.5 GiB", "高精度，需要较多内存"},
	{"large-v3-turbo-q5_0", "547 MiB", "大模型量化版，推荐用于生产"},
	{"large-v3-turbo", "1.5 GiB", "大模型 turbo，速度与质量最佳平衡"},
	{"large-v3", "2.9 GiB", "最高精度，需要充足内存"},
}

// modelDownloadStatus 模型下载状态，内存中追踪
type modelDownloadStatus struct {
	mu       sync.Mutex
	active   map[string]bool
	progress map[string]int
}

var dlStatus = &modelDownloadStatus{
	active:   make(map[string]bool),
	progress: make(map[string]int),
}

var runtimeInstallMu sync.Mutex
var runtimeInstalling bool

// start 记录模型开始下载，供多个请求读取一致状态。
func (s *modelDownloadStatus) start(name string) {
	s.mu.Lock()
	s.active[name] = true
	s.progress[name] = 0
	s.mu.Unlock()
}

// finish 清理已结束的下载状态，文件状态由磁盘结果决定。
func (s *modelDownloadStatus) finish(name string) {
	s.mu.Lock()
	delete(s.active, name)
	delete(s.progress, name)
	s.mu.Unlock()
}

// setProgress 保存模型下载百分比。
func (s *modelDownloadStatus) setProgress(name string, p int) {
	s.mu.Lock()
	s.progress[name] = p
	s.mu.Unlock()
}

// isActive 判断指定模型是否正在下载。
func (s *modelDownloadStatus) isActive(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active[name]
}

// getProgress 返回指定模型当前下载百分比。
func (s *modelDownloadStatus) getProgress(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress[name]
}

// modelsDir 返回模型存储目录 ~/dsub/models/
func modelsDir() string {
	return filepath.Join(conf.DataDir(), "models")
}

// modelPath 返回指定模型文件的完整路径
func modelPath(name string) string {
	return filepath.Join(modelsDir(), fmt.Sprintf("ggml-%s.bin", name))
}

// RegisterModel 注册 Whisper 运行时与模型管理路由。
func RegisterModel(r gin.IRouter, hub ws.Huber, cfg *conf.Bootstrap) {
	g := r.Group("/models")
	g.GET("", web.WrapH(listModels))
	g.POST("/download", web.WrapH(func(c *gin.Context, in *modelDownloadInput) (any, error) {
		return startDownload(in.Name, hub)
	}))
	g.GET("/runtime", web.WrapH(func(_ *gin.Context, _ *struct{}) (any, error) {
		return inspectWhisperRuntime(cfg), nil
	}))
	g.POST("/runtime/install", web.WrapH(func(_ *gin.Context, _ *struct{}) (any, error) {
		return startRuntimeInstall(hub, cfg)
	}))
}

type whisperRuntimeOutput struct {
	whisperadapter.RuntimeInfo
	Installing bool `json:"installing"`
}

// inspectWhisperRuntime 返回真实运行时状态，避免界面使用固定假数据。
func inspectWhisperRuntime(cfg *conf.Bootstrap) whisperRuntimeOutput {
	runtimeInstallMu.Lock()
	installing := runtimeInstalling
	runtimeInstallMu.Unlock()
	return whisperRuntimeOutput{
		RuntimeInfo: whisperadapter.InspectRuntime(cfg.Pipeline.WhisperBin),
		Installing:  installing,
	}
}

// startRuntimeInstall 启动唯一安装任务，并把安装输出推送到界面。
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

		err := whisperadapter.InstallRuntime(context.Background(), func(line string) {
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

// listModels 返回模型列表及下载状态
func listModels(_ *gin.Context, _ *struct{}) ([]modelListOutput, error) {
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
	Name string `json:"name" binding:"required"`
}

// startDownload 启动后台模型下载，通过 WebSocket 推送进度
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

// downloadModel 从 HuggingFace 下载 ggml 模型文件
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
