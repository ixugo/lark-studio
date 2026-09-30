package wails

import (
	"crypto/rand"
	_ "embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	youtubeadapter "github.com/ixugo/vdub/internal/adapter/youtube"
	"github.com/ixugo/vdub/internal/core/pipeline"
	"github.com/ixugo/vdub/internal/core/task"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed youtube_verify.js
var youtubeVerifyJS string

type youtubeVerification struct {
	window  application.Window
	nonce   string
	timer   *time.Timer
	pending *youtubePending
}

type youtubePending struct {
	link   string
	height int
	ffmpeg string
	recipe *task.CreateTaskInput
}

func (s *AppService) youtubeManager() *youtubeadapter.Manager {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.youtube == nil {
		s.youtube = youtubeadapter.NewManager()
	}
	return s.youtube
}
func (s *AppService) ResolveYouTubeVideo(link string) (*youtubeadapter.Info, error) {
	return s.youtubeManager().Inspect(link)
}
func (s *AppService) GetYouTubeDownload() (youtubeadapter.Status, error) {
	status := s.youtubeManager().Status()
	if status.Directory == "" {
		resolved, err := youtubeadapter.DownloadDirectory("~/Documents/lark-studio")
		if err != nil {
			return status, err
		}
		status.Directory = resolved
	}
	return status, nil
}
func (s *AppService) StartYouTubeDownload(link string, height int) error {
	return s.StartYouTubeDownloadWithRecipe(link, height, nil)
}
func (s *AppService) StartYouTubeDownloadWithRecipe(link string, height int, recipe *task.CreateTaskInput) error {
	if recipe != nil {
		if recipe.Mode < pipeline.ModeSubtitle || recipe.Mode > pipeline.ModeDirectDub {
			return errors.New("请选择有效的视频配方")
		}
		if err := pipeline.ValidateResourceMode("video.mp4", recipe.Mode); err != nil {
			return err
		}
		copy := *recipe
		copy.InputPath = ""
		copy.OutputDir = ""
		recipe = &copy
	}
	normalized, err := youtubeadapter.NormalizeURL(link)
	if err != nil {
		return err
	}
	manager := s.youtubeManager()
	status := manager.Status()
	if status.Video == nil || status.Video.URL != normalized || !slices.Contains(status.Video.Resolutions, height) {
		return errors.New("请先解析链接并选择分辨率")
	}
	s.mu.RLock()
	ffmpeg := s.bc.Pipeline.FFmpegBin
	s.mu.RUnlock()
	return s.openYouTubeVerification(normalized, &youtubePending{link: normalized, height: height, ffmpeg: ffmpeg, recipe: recipe})
}
func (s *AppService) CancelYouTubeDownload() {
	s.closeYouTubeVerification()
	s.youtubeManager().Cancel()
}
func (s *AppService) VerifyYouTubeDownload(link string) error {
	return s.openYouTubeVerification(link, nil)
}
func (s *AppService) openYouTubeVerification(link string, pending *youtubePending) error {
	normalized, err := youtubeadapter.NormalizeURL(link)
	if err != nil {
		return err
	}
	manager := s.youtubeManager()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app == nil {
		return errors.New("请在桌面版完成下载服务校验")
	}
	if s.youtubeVerification.window != nil {
		s.youtubeVerification.window.Focus()
		return nil
	}
	if err = manager.BeginVerify(); err != nil {
		return err
	}
	nonce := rand.Text()
	encoded, err := json.Marshal(nonce)
	if err != nil {
		manager.EndVerify()
		return err
	}
	id := strings.TrimPrefix(normalized, "https://www.youtube.com/watch?v=")
	encodedID, err := json.Marshal(id)
	if err != nil {
		return err
	}
	quality := 1080
	if pending != nil {
		quality = pending.height
	}
	script := strings.NewReplacer("__NONCE__", string(encoded), "__VIDEO_ID__", string(encodedID), "__QUALITY__", fmt.Sprint(quality)).Replace(youtubeVerifyJS)
	window := s.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "下载服务校验 / Download verification", Width: 760, Height: 700, MinWidth: 600, MinHeight: 500,
		URL:             youtubeadapter.ServiceOrigin + "/v1/full?videoId=" + id,
		JS:              script,
		InitialPosition: application.WindowCentered,
	})
	s.youtubeVerification = youtubeVerification{window: window, nonce: nonce, pending: pending}
	s.youtubeVerification.timer = time.AfterFunc(2*time.Minute, func() { s.closeYouTubeVerificationID(window.ID()) })
	window.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) { s.releaseYouTubeVerification(window.ID()) })
	return nil
}
func (s *AppService) releaseYouTubeVerification(id uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.youtubeVerification
	if v.window == nil || v.window.ID() != id {
		return
	}
	if v.timer != nil {
		v.timer.Stop()
	}
	s.youtubeVerification = youtubeVerification{}
	if s.youtube != nil {
		s.youtube.EndVerify()
	}
}
func (s *AppService) closeYouTubeVerification() {
	s.mu.RLock()
	window := s.youtubeVerification.window
	s.mu.RUnlock()
	if window != nil {
		s.closeYouTubeVerificationID(window.ID())
	}
}
func (s *AppService) closeYouTubeVerificationID(id uint) {
	s.mu.Lock()
	v := s.youtubeVerification
	if v.window == nil || v.window.ID() != id {
		s.mu.Unlock()
		return
	}
	if v.timer != nil {
		v.timer.Stop()
	}
	s.youtubeVerification = youtubeVerification{}
	if s.youtube != nil {
		s.youtube.EndVerify()
	}
	s.mu.Unlock()
	v.window.Close()
}

func validYouTubeOrigin(origin *application.OriginInfo) bool {
	if origin == nil {
		return false
	}
	u, err := url.Parse(origin.Origin)
	if err != nil || u.Scheme != "https" || u.Hostname() != "embed.dlsrv.online" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	// macOS 提供主框架标记，Windows 提供发送页面与顶层页面的 URL。
	return origin.IsMainFrame || (origin.TopOrigin != "" && origin.Origin == origin.TopOrigin)
}
func (s *AppService) handleYouTubeMessage(window application.Window, message string, origin *application.OriginInfo) {
	if window == nil || !validYouTubeOrigin(origin) || len(message) > 12000 {
		return
	}
	var payload struct {
		Kind      string `json:"kind"`
		Nonce     string `json:"nonce"`
		Token     string `json:"token"`
		UserAgent string `json:"user_agent"`
		Address   string `json:"address"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal([]byte(message), &payload); err != nil || (payload.Kind != "youtube-session" && payload.Kind != "youtube-error") {
		return
	}
	var session youtubeadapter.Session
	if payload.Kind == "youtube-session" {
		parsed, err := youtubeadapter.ParseSession(payload.Token, payload.UserAgent)
		if err != nil {
			return
		}
		session = parsed
		if err := youtubeadapter.ValidateDownloadAddress(payload.Address); err != nil {
			return
		}
	}
	s.mu.Lock()
	v, manager := s.youtubeVerification, s.youtube
	if v.window == nil || v.window.ID() != window.ID() || payload.Nonce != v.nonce || manager == nil {
		s.mu.Unlock()
		return
	}
	if v.timer != nil {
		v.timer.Stop()
	}
	s.youtubeVerification = youtubeVerification{}
	if payload.Kind == "youtube-error" {
		manager.EndVerify()
		manager.Fail(fmt.Errorf("%.500s", payload.Error))
	} else if manager.AcceptSession(session) && v.pending != nil {
		plan := youtubeadapter.DownloadPlan{Address: payload.Address}
		if v.pending.recipe != nil {
			recipe := *v.pending.recipe
			plan.OnComplete = func(path string) (string, error) {
				recipe.InputPath = path
				created, err := s.CreateTask(recipe)
				if err != nil {
					return "", err
				}
				return created.ID, nil
			}
		}
		if err := manager.Start(v.pending.link, v.pending.height, "~/Documents/lark-studio", v.pending.ffmpeg, plan); err != nil {
			manager.Fail(err)
		}
	}
	s.mu.Unlock()
	v.window.Close()
}

func (s *AppService) ResetYouTubeDownload() error {
	return s.youtubeManager().Reset()
}

func (s *AppService) GetYouTubeStorage() (youtubeadapter.Storage, error) {
	dir, err := youtubeadapter.DownloadDirectory("~/Documents/lark-studio")
	if err != nil {
		return youtubeadapter.Storage{}, err
	}
	return s.youtubeManager().Storage(dir)
}

func (s *AppService) ClearYouTubeDownloads() error {
	dir, err := youtubeadapter.DownloadDirectory("~/Documents/lark-studio")
	if err != nil {
		return err
	}
	return s.youtubeManager().ClearStorage(dir)
}
