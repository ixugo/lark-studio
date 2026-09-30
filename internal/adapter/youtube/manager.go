package youtube

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Info struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Resolutions []int  `json:"resolutions"`
}
type Status struct {
	Video     *Info   `json:"video,omitempty"`
	Phase     string  `json:"phase"`
	Percent   float64 `json:"percent"`
	Path      string  `json:"path"`
	Error     string  `json:"error"`
	Directory string  `json:"directory"`
	Verified  bool    `json:"verified"`
	Bytes     int64   `json:"bytes"`
	Total     int64   `json:"total"`
}
type Manager struct {
	mu       sync.Mutex
	status   Status
	session  Session
	cancel   context.CancelFunc
	info     *Info
	http     *http.Client
	endpoint string
}

func NewManager() *Manager {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 60 * time.Second
	client := &http.Client{Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("下载重定向过多")
		}
		return validateTunnel(req.URL.String())
	}}
	return &Manager{status: Status{Phase: "idle"}, http: client, endpoint: ServiceOrigin}
}
func busy(phase string) bool {
	return slices.Contains([]string{"verifying", "inspecting", "converting", "downloading", "checking"}, phase)
}
func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := m.status
	result.Video = m.info
	result.Verified = time.Until(m.session.Expires) > 10*time.Second
	return result
}
func (m *Manager) BeginVerify() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if busy(m.status.Phase) {
		return errors.New("请等待当前操作完成")
	}
	m.session = Session{}
	m.status = Status{Phase: "verifying"}
	return nil
}
func (m *Manager) AcceptSession(session Session) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.Phase != "verifying" {
		return false
	}
	m.session = session
	m.status.Phase = "idle"
	return true
}
func (m *Manager) EndVerify() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.Phase == "verifying" {
		m.status.Phase = "idle"
	}
}
func (m *Manager) set(fn func(*Status)) { m.mu.Lock(); defer m.mu.Unlock(); fn(&m.status) }
func (m *Manager) Inspect(raw string) (*Info, error) {
	link, err := NormalizeURL(raw)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if busy(m.status.Phase) {
		m.mu.Unlock()
		return nil, errors.New("请等待当前操作完成")
	}
	m.status = Status{Phase: "inspecting"}
	m.info = nil
	m.mu.Unlock()
	defer m.set(func(s *Status) { s.Phase = "idle" })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	address := "https://www.youtube.com/oembed?url=" + url.QueryEscape(link) + "&format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return nil, errors.New("无法读取视频信息，请检查链接及网络")
	}
	var metadata struct {
		Title string `json:"title"`
	}
	if err = readJSON(resp, &metadata); err != nil {
		return nil, fmt.Errorf("读取视频信息失败：%w", err)
	}
	if metadata.Title == "" {
		return nil, errors.New("视频标题为空，请检查链接")
	}
	info := &Info{URL: link, Title: metadata.Title, Resolutions: []int{1080, 720, 480, 360, 240, 144}}
	m.mu.Lock()
	m.info = info
	m.mu.Unlock()
	return info, nil
}
func (m *Manager) Start(raw string, height int, outputDir, ffmpeg string) error {
	link, err := NormalizeURL(raw)
	if err != nil {
		return err
	}
	dir, err := DownloadDirectory(outputDir)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if busy(m.status.Phase) {
		return errors.New("请等待当前操作完成")
	}
	if m.info == nil || m.info.URL != link || !slices.Contains(m.info.Resolutions, height) {
		return errors.New("请先解析链接并选择分辨率")
	}
	if time.Until(m.session.Expires) < 10*time.Second {
		return errors.New("请先完成下载服务校验")
	}
	ffmpeg, err = findFFmpeg(ffmpeg)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	info, session := *m.info, m.session
	path := filepath.Join(dir, fmt.Sprintf("%s [%s] [%dp].mp4", safeTitle(info.Title), videoIDFromURL(link), height))
	if _, err = os.Stat(path); err == nil {
		return errors.New("该分辨率的视频已存在，不覆盖已有文件")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	m.cancel = cancel
	m.status = Status{Phase: "converting", Directory: dir}
	go func() {
		defer cancel()
		err := m.download(ctx, info, height, session, path, ffmpeg)
		m.finish(ctx, path, err)
	}()
	return nil
}
func (m *Manager) download(ctx context.Context, info Info, height int, session Session, path, ffmpeg string) error {
	conversionCtx, stop := context.WithTimeout(ctx, 2*time.Minute)
	address, err := m.convert(conversionCtx, info.URL, height, session)
	stop()
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".youtube-*.part")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer func() {
		if err := os.Remove(temp); err != nil && !errors.Is(err, os.ErrNotExist) {
			m.set(func(s *Status) { s.Error = "临时文件清理失败：" + temp })
		}
	}()
	err = m.saveMedia(ctx, address, session.UserAgent, file)
	err = errors.Join(err, file.Close())
	if err != nil {
		return err
	}
	m.set(func(s *Status) { s.Phase = "checking"; s.Percent = 99 })
	if err = checkMedia(ctx, ffmpeg, temp, height); err != nil {
		return err
	}
	if err = os.Link(temp, path); err != nil {
		return fmt.Errorf("保存视频失败：%w", err)
	}
	return nil
}
func (m *Manager) saveMedia(ctx context.Context, address, agent string, file *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", agent)
	resp, err := m.http.Do(req)
	if err != nil {
		return errors.New("视频下载连接失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("视频下载失败（HTTP %d），请重新校验后重试", resp.StatusCode)
	}
	if resp.ContentLength > maxVideoBytes {
		return errors.New("视频超过 20 GiB 下载上限")
	}
	m.set(func(s *Status) { s.Phase = "downloading"; s.Total = resp.ContentLength })
	buffer := make([]byte, 128<<10)
	var received int64
	for {
		n, readErr := resp.Body.Read(buffer)
		if n > 0 {
			received += int64(n)
			if received > maxVideoBytes {
				return errors.New("视频超过 20 GiB 下载上限")
			}
			if _, err = file.Write(buffer[:n]); err != nil {
				return err
			}
			m.set(func(s *Status) {
				s.Bytes = received
				if s.Total > 0 {
					s.Percent = min(98, 100*float64(received)/float64(s.Total))
				}
			})
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return errors.New("视频下载中断，请重试")
		}
	}
	if received == 0 || (resp.ContentLength >= 0 && received != resp.ContentLength) {
		return errors.New("下载的视频不完整")
	}
	return nil
}
func (m *Manager) finish(ctx context.Context, path string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancel = nil
	if err != nil {
		m.status.Phase = "failed"
		m.status.Error = err.Error()
		if ctx.Err() != nil {
			m.status.Error = "操作已取消或超时"
			if errors.Is(ctx.Err(), context.Canceled) {
				m.status.Phase = "cancelled"
			}
		}
		return
	}
	m.status.Phase = "completed"
	m.status.Path = path
	m.status.Percent = 100
}
func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
}
func findFFmpeg(configured string) (string, error) {
	if configured != "" && configured != "ffmpeg" {
		p, err := exec.LookPath(configured)
		if err != nil {
			return "", errors.New("配置的 FFmpeg 路径不可用")
		}
		return p, nil
	}
	for _, p := range []string{"ffmpeg", "/opt/homebrew/bin/ffmpeg", "/usr/local/bin/ffmpeg"} {
		if found, err := exec.LookPath(p); err == nil {
			return found, nil
		}
	}
	return "", errors.New("请先在设置中配置 FFmpeg 路径")
}
func safeTitle(title string) string {
	title = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, title)
	runes := []rune(strings.Trim(title, " ."))
	if len(runes) > 60 {
		runes = runes[:60]
	}
	if len(runes) == 0 {
		return "YouTube"
	}
	return string(runes)
}

var videoDimensions = regexp.MustCompile(`Video:.*?\b[0-9]{2,5}x([0-9]{2,5})\b`)

func checkMedia(ctx context.Context, ffmpeg, path string, height int) error {
	cmd := exec.CommandContext(ctx, ffmpeg, "-nostdin", "-hide_banner", "-i", path, "-map", "0:v:0", "-map", "0:a:0", "-t", "0.01", "-c", "copy", "-f", "null", "-")
	cmd.WaitDelay = 3 * time.Second
	data, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("下载结果不是有效的音视频文件")
	}
	match := videoDimensions.FindSubmatch(data)
	if len(match) != 2 {
		return errors.New("无法确认下载视频的分辨率")
	}
	actual, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return err
	}
	if actual != height {
		return fmt.Errorf("服务返回 %dp，未达到所选 %dp，已取消保存", actual, height)
	}
	return nil
}

func (m *Manager) Fail(err error) {
	m.set(func(s *Status) { s.Phase = "failed"; s.Error = err.Error() })
}
