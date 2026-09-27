package api

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

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
	{"large-v3-turbo", "1.62 GiB", "旗舰加速推荐，多语言高精度识别"},
	{"tiny", "75 MiB", "最小型多语言模型，适合轻量转写与设备测试"},
	{"medium", "1.53 GiB", "多语中型模型，中文识别效果佳"},
	{"small", "488 MiB", "多语轻量模型，速度与质量均衡"},
	{"medium.en", "1.53 GiB", "英文专用高质量模型，英语转录首选"},
	{"small.en", "488 MiB", "英文专用轻量模型，速度快且精度高"},
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
	dir := filepath.Join(conf.StudioDir(), "models")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// ModelPath 返回指定模型在 ~/.lark-studio/models 下的完整路径。
func ModelPath(name string) string {
	return filepath.Join(modelsDir(), fmt.Sprintf("ggml-%s.bin", name))
}

func modelPath(name string) string {
	return ModelPath(name)
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

// ModelListOutput 描述 GGML 模型或 VAD 资源的管理状态。
type ModelListOutput struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Size        string `json:"size"`
	Desc        string `json:"desc"`
	Downloaded  bool   `json:"downloaded"`
	Path        string `json:"path,omitempty"`
	Downloading bool   `json:"downloading"`
	Progress    int    `json:"progress"`
}

func listModels(_ *http.Request, _ *struct{}) ([]ModelListOutput, error) {
	out := make([]ModelListOutput, 0, len(whisperModels))
	for _, m := range whisperModels {
		p := modelPath(m.Name)
		_, err := os.Stat(p)
		downloaded := err == nil
		item := ModelListOutput{
			Name:        m.Name,
			Kind:        "asr",
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
	return discoverInstalledModels(out)
}

// discoverInstalledModels 将管理目录中的所有 GGML 文件并入推荐模型清单。
func discoverInstalledModels(models []ModelListOutput) ([]ModelListOutput, error) {
	known := make(map[string]int, len(models))
	for i, model := range models {
		known[model.Name] = i
	}
	extraModels := make([]ModelListOutput, 0)
	extraIndex := make(map[string]int)
	if err := filepath.WalkDir(modelsDir(), func(path string, entry fs.DirEntry, walkErr error) error {
		return collectInstalledModel(path, entry, walkErr, models, known, &extraModels, extraIndex)
	}); err != nil {
		return nil, fmt.Errorf("扫描语音模型目录失败: %w", err)
	}
	slices.SortFunc(extraModels, func(a, b ModelListOutput) int { return cmp.Compare(a.Name, b.Name) })
	return append(models, extraModels...), nil
}

// collectInstalledModel 识别单个资源文件并更新推荐项或补充本地模型。
func collectInstalledModel(path string, entry fs.DirEntry, walkErr error, models []ModelListOutput, known map[string]int, extras *[]ModelListOutput, extraIndex map[string]int) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.IsDir() || !strings.HasPrefix(entry.Name(), "ggml-") || !strings.HasSuffix(entry.Name(), ".bin") {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	name := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "ggml-"), ".bin")
	if index, ok := known[name]; ok {
		models[index].Downloaded = true
		models[index].Path = path
		models[index].Size = formatModelSize(info.Size())
		return nil
	}
	if index, ok := extraIndex[name]; ok {
		(*extras)[index].Path = path
		return nil
	}
	kind, desc := "asr", "已发现的本地 GGML 语音模型"
	if strings.HasPrefix(name, "silero-") {
		kind, desc = "vad", "Silero 人声检测模型"
	}
	extraIndex[name] = len(*extras)
	*extras = append(*extras, ModelListOutput{Name: name, Kind: kind, Size: formatModelSize(info.Size()), Desc: desc, Downloaded: true, Path: path})
	return nil
}

// formatModelSize 以易读单位展示已安装模型的实际文件大小。
func formatModelSize(size int64) string {
	const mib = 1 << 20
	if size >= mib {
		return fmt.Sprintf("%.1f MiB", float64(size)/mib)
	}
	return fmt.Sprintf("%d KiB", (size+1023)/1024)
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
	candidates := []string{
		fmt.Sprintf("https://hf-mirror.com/ggerganov/whisper.cpp/resolve/main/ggml-%s.bin", name),
		fmt.Sprintf("https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-%s.bin", name),
	}
	url := whisperadapter.ProbeFastestURL(context.Background(), candidates)
	dest := modelPath(name)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("创建模型目录失败: %w", err)
	}

	tmpFile := dest + ".tmp"
	var existingSize int64
	if fi, err := os.Stat(tmpFile); err == nil {
		existingSize = fi.Size()
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("构造下载请求失败: %w", err)
	}

	isRange := false
	if existingSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingSize))
		isRange = true
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	var f *os.File
	var total int64
	var written int64 = existingSize

	if isRange && resp.StatusCode == http.StatusPartialContent {
		// 服务器支持断点续传
		total = existingSize + resp.ContentLength
		f, err = os.OpenFile(tmpFile, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("打开断点文件失败: %w", err)
		}
	} else if resp.StatusCode == http.StatusOK {
		total = resp.ContentLength
		written = 0
		f, err = os.Create(tmpFile)
		if err != nil {
			return fmt.Errorf("创建临时文件失败: %w", err)
		}
	} else {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	buf := make([]byte, 128*1024)
	lastPct := -1
	startTime := time.Now()
	lastReportTime := startTime
	var lastReportWritten int64 = written

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := f.Write(buf[:n]); wErr != nil {
				f.Close()
				return fmt.Errorf("写入失败: %w", wErr)
			}
			written += int64(n)

			now := time.Now()
			if now.Sub(lastReportTime) >= 300*time.Millisecond || readErr == io.EOF {
				pct := 0
				if total > 0 {
					pct = int(written * 100 / total)
				}
				sec := now.Sub(lastReportTime).Seconds()
				var speedStr string
				if sec > 0 {
					speedBytes := float64(written-lastReportWritten) / sec
					if speedBytes >= 1024*1024 {
						speedStr = fmt.Sprintf("%.1f MB/s", speedBytes/(1024*1024))
					} else {
						speedStr = fmt.Sprintf("%.0f KB/s", speedBytes/1024)
					}
				}
				lastReportTime = now
				lastReportWritten = written

				if pct != lastPct || speedStr != "" {
					lastPct = pct
					dlStatus.setProgress(name, pct)
					if hub != nil {
						hub.Broadcast(ws.NewMessage("model_download_progress", map[string]any{
							"model":      name,
							"progress":   pct,
							"speed":      speedStr,
							"downloaded": written,
							"total":      total,
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
			return fmt.Errorf("读取失败: %w", readErr)
		}
	}
	f.Close()

	if err := os.Rename(tmpFile, dest); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}

	slog.Info("model downloaded", "model", name, "path", dest, "size", written)
	if hub != nil {
		hub.Broadcast(ws.NewMessage("model_download_done", map[string]any{
			"model": name, "path": dest, "size": written,
		}))
	}
	return nil
}

// ListAllWhisperModels 供后端与 Wails 服务调用的模型列表数据。
func ListAllWhisperModels() []ModelListOutput {
	list, _ := listModels(nil, nil)
	return list
}

// StartWhisperModelDownload 供 Wails 服务调用的启动下载模型。
func StartWhisperModelDownload(name string, hub ws.Huber) (any, error) {
	return startDownload(name, hub)
}

// DeleteWhisperModelFile 从统一模型目录删除指定 GGML 文件。
func DeleteWhisperModelFile(name string) error {
	p, err := findModelFile(name)
	if err != nil {
		return err
	}
	_ = os.Remove(p + ".tmp")
	return os.Remove(p)
}

// findModelFile 仅在模型目录中按 GGML 文件名查找，避免删除任意路径。
func findModelFile(name string) (string, error) {
	want := fmt.Sprintf("ggml-%s.bin", name)
	var found string
	err := filepath.WalkDir(modelsDir(), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && entry.Name() == want {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("查找语音模型失败: %w", err)
	}
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
}
