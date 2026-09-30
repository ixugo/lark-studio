package youtube

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	for _, tc := range []struct {
		link  string
		valid bool
	}{
		{"https://www.youtube.com/watch?v=0kILa02vKuI&list=PLtest&index=3", true},
		{"https://youtu.be/0kILa02vKuI", true},
		{"https://youtube.com/shorts/0kILa02vKuI", true},
		{"https://youtube.com.evil.test/watch?v=0kILa02vKuI", false},
		{"https://user@youtube.com/watch?v=0kILa02vKuI", false},
		{"https://youtube.com:8080/watch?v=0kILa02vKuI", false},
		{"https://youtube.com/playlist?list=test", false},
		{strings.Repeat("a", 2049), false},
	} {
		got, err := NormalizeURL(tc.link)
		if (err == nil) != tc.valid {
			t.Fatalf("link=%q got=%q err=%v", tc.link, got, err)
		}
		if tc.valid && got != "https://www.youtube.com/watch?v=0kILa02vKuI" {
			t.Fatal(got)
		}
	}
}
func TestDownloadDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := DownloadDirectory("~/Documents/lark-studio")
	if err != nil || got != filepath.Join(home, "Documents/lark-studio", "Downloads") {
		t.Fatalf("got=%s err=%v", got, err)
	}
	if _, err = DownloadDirectory(""); err == nil {
		t.Fatal("empty directory accepted")
	}
}
func TestTunnelURL(t *testing.T) {
	for _, tc := range []struct {
		url   string
		valid bool
	}{
		{"https://yt1s-worker-5.dlsrv.online/tunnel?id=abc", true},
		{"https://evil.test/tunnel", false},
		{"http://yt1s-worker-5.dlsrv.online/tunnel", false},
		{"https://user@yt1s-worker-5.dlsrv.online/tunnel", false},
		{"https://yt1s-worker-5.dlsrv.online:8080/tunnel", false},
		{"https://yt1s-worker-5.dlsrv.online/other", false},
	} {
		if err := validateTunnel(tc.url); (err == nil) != tc.valid {
			t.Fatalf("%s: %v", tc.url, err)
		}
	}
}

func sessionToken(expiry time.Time) string {
	data := fmt.Sprintf(`{"exp":%d}`, expiry.Unix())
	return "header." + base64.RawURLEncoding.EncodeToString([]byte(data)) + ".signature"
}
func TestSessionAndBusyState(t *testing.T) {
	m := NewManager()
	token := sessionToken(time.Now().Add(5 * time.Minute))
	session, err := ParseSession(token, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	if m.AcceptSession(session) {
		t.Fatal("accepted unsolicited session")
	}
	if err = m.BeginVerify(); err != nil {
		t.Fatal(err)
	}
	if err = m.BeginVerify(); err == nil {
		t.Fatal("duplicate verification accepted")
	}
	if !m.AcceptSession(session) || !m.Status().Verified {
		t.Fatal(m.Status())
	}
	if _, err = ParseSession(sessionToken(time.Now().Add(-time.Minute)), "test-agent"); err == nil {
		t.Fatal("expired session accepted")
	}
	if _, err = ParseSession(token, "agent\nInjected: value"); err == nil {
		t.Fatal("header injection accepted")
	}
	if err = m.Start("https://youtu.be/0kILa02vKuI", 1080, t.TempDir(), ""); err == nil {
		t.Fatal("download without resolved metadata accepted")
	}
}

func TestConversionUsesAuthenticatedExactQuality(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/download/mp4" || r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Sec-Fetch-Site") != "same-origin" || r.Header.Get("User-Agent") != "test-agent" {
			t.Errorf("request not authenticated: %s %s", r.Method, r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != `{"videoId":"0kILa02vKuI","format":"mp4","quality":"1080"}` {
			t.Errorf("wrong selection: %s", body)
		}
		if _, err = w.Write([]byte(`{"url":"https://yt1s-worker-5.dlsrv.online/tunnel?id=fixture"}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	m := NewManager()
	m.endpoint = server.URL
	link, err := m.convert(t.Context(), "https://www.youtube.com/watch?v=0kILa02vKuI", 1080, Session{Token: "test-token", UserAgent: "test-agent"})
	if err != nil || link != "https://yt1s-worker-5.dlsrv.online/tunnel?id=fixture" {
		t.Fatalf("link=%s err=%v", link, err)
	}
}
func TestDownloadRejectsTruncatedMediaAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		if _, err := w.Write([]byte("short")); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	m := NewManager()
	file, err := os.CreateTemp(t.TempDir(), "video")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err = m.saveMedia(t.Context(), server.URL, "test-agent", file); err == nil {
		t.Fatal("truncated media accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err = m.saveMedia(ctx, server.URL, "test-agent", file); err == nil {
		t.Fatal("cancelled media accepted")
	}
}
func TestMediaValidationChecksActualHeightAndAudio(t *testing.T) {
	ffmpeg, err := findFFmpeg("")
	if err != nil {
		t.Skip(err)
	}
	path := filepath.Join(t.TempDir(), "fixture.mp4")
	cmd := exec.CommandContext(t.Context(), ffmpeg, "-nostdin", "-v", "error", "-f", "lavfi", "-i", "color=s=256x144:d=0.2", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo", "-shortest", "-c:v", "libx264", "-c:a", "aac", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fixture failed: %v %s", err, output)
	}
	if err = checkMedia(t.Context(), ffmpeg, path, 144); err != nil {
		t.Fatal(err)
	}
	if err = checkMedia(t.Context(), ffmpeg, path, 1080); err == nil {
		t.Fatal("lower quality accepted as 1080p")
	}
	invalid := filepath.Join(t.TempDir(), "invalid.mp4")
	if err = os.WriteFile(invalid, []byte("<html>not video</html>"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = checkMedia(t.Context(), ffmpeg, invalid, 144); err == nil {
		t.Fatal("HTML accepted as media")
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestStartPublishesMediaInDefaultDirectory(t *testing.T) {
	ffmpeg, err := findFFmpeg("")
	if err != nil {
		t.Skip(err)
	}
	fixture := filepath.Join(t.TempDir(), "fixture.mp4")
	if data, err := exec.CommandContext(t.Context(), ffmpeg, "-nostdin", "-v", "error", "-f", "lavfi", "-i", "color=s=256x144:d=0.2", "-f", "lavfi", "-i", "anullsrc=r=44100:cl=stereo", "-shortest", "-c:v", "libx264", "-c:a", "aac", fixture).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, data)
	}
	media, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	defer close(release)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/download/mp4" {
			if _, err := w.Write([]byte(`{"url":"https://yt1s-worker-5.dlsrv.online/tunnel?id=test"}`)); err != nil {
				t.Error(err)
			}
			return
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(media)))
		if _, err := w.Write(media); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager()
	transport := m.http.Transport
	m.http.Transport = testTransport(func(req *http.Request) (*http.Response, error) {
		copy := req.Clone(req.Context())
		copy.URL = req.URL.Clone()
		copy.URL.Scheme = target.Scheme
		copy.URL.Host = target.Host
		return transport.RoundTrip(copy)
	})
	m.endpoint = server.URL
	m.info = &Info{URL: "https://www.youtube.com/watch?v=0kILa02vKuI", Title: "../Example: video", Resolutions: []int{144}}
	m.session = Session{Token: "token", UserAgent: "agent", Expires: time.Now().Add(5 * time.Minute)}
	dir := t.TempDir()
	if err = m.Start(m.info.URL, 144, dir, ffmpeg); err != nil {
		t.Fatal(err)
	}
	if err = m.Start(m.info.URL, 144, dir, ffmpeg); err == nil {
		t.Fatal("concurrent download accepted")
	}
	release <- struct{}{}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		status := m.Status()
		if status.Phase == "completed" {
			expected := filepath.Join(dir, "Downloads", "_Example_ video [0kILa02vKuI] [144p].mp4")
			if status.Path != expected {
				t.Fatalf("unexpected download path: %s", status.Path)
			}
			got, err := os.ReadFile(status.Path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, media) {
				t.Fatal("media bytes differ")
			}
			if err = m.Start(m.info.URL, 144, dir, ffmpeg); err == nil {
				t.Fatal("existing file overwrite accepted")
			}
			files, err := filepath.Glob(filepath.Join(dir, "Downloads", ".youtube-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 0 {
				t.Fatalf("temporary files left: %v", files)
			}
			return
		}
		if status.Phase == "failed" {
			t.Fatal(status.Error)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("download did not complete")
}
