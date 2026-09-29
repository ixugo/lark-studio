package update

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func testClient(status int, body string) *Client {
	c := NewClient()
	c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})
	return c
}

const releaseJSON = `{"tag_name":"v0.1.0","updated_at":"2020-01-01T00:00:00Z","body":"更新说明","assets":[{"name":"lark-studio_v0.0.127_macos_arm64.dmg","size":3,"browser_download_url":"https://github.com/ixugo/lark-studio/releases/download/v0.1.0/lark-studio_v0.0.127_macos_arm64.dmg"}]}`

func TestFetchRelease(t *testing.T) {
	r, err := testClient(200, releaseJSON).Fetch(t.Context(), "darwin", "arm64")
	if err != nil || r.Version != "v0.1.0" || r.Notes != "更新说明" || !r.Supported() {
		t.Fatalf("release=%+v,err=%v", r, err)
	}
	r, err = testClient(200, releaseJSON).Fetch(t.Context(), "linux", "amd64")
	if err != nil || r.Supported() || r.Reason() == "" {
		t.Fatalf("unsupported release=%+v,err=%v", r, err)
	}
}

func TestRedirectAllowlist(t *testing.T) {
	for _, address := range []string{"http://github.com/x", "https://github.com.evil.example/x", "https://user@github.com/x", "https://github.com:443/x", "https://evil.example/x"} {
		u, err := url.Parse(address)
		if err != nil {
			t.Fatal(err)
		}
		if err := trustedRedirect(&http.Request{URL: u}, nil); err == nil {
			t.Errorf("accepted redirect %s", address)
		}
	}
	u, err := url.Parse("https://release-assets.githubusercontent.com/x?token=opaque")
	if err != nil {
		t.Fatal(err)
	}
	if err := trustedRedirect(&http.Request{URL: u}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestFetchRejectsInvalidResponse(t *testing.T) {
	for _, tt := range []struct {
		status int
		body   string
	}{
		{403, `{}`}, {200, `broken`}, {200, `{"tag_name":"dev"}`}, {200, `{"tag_name":"1.2.3","body":22}`},
		{200, strings.ReplaceAll(releaseJSON, "https://github.com/ixugo/lark-studio/releases/download/", "https://evil.example/")},
		{200, strings.ReplaceAll(releaseJSON, `"size":3`, `"size":-1`)},
		{200, releaseJSON + `{}`},
	} {
		if _, err := testClient(tt.status, tt.body).Fetch(t.Context(), "darwin", "arm64"); err == nil {
			t.Errorf("accepted %+v", tt)
		}
	}
}

func TestDownloadChecksSizeAndDigest(t *testing.T) {
	for _, tt := range []struct {
		body, digest string
		size         int64
		valid        bool
	}{
		{"abc", fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("abc"))), 3, true},
		{"abc", fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("xyz"))), 3, false},
		{"abcd", "", 3, false}, {"ab", "", 3, false},
	} {
		c := testClient(200, tt.body)
		path := t.TempDir() + "/asset"
		err := c.download(t.Context(), asset{Name: "asset", URL: "https://github.com/ixugo/lark-studio/releases/download/v1.0.0/asset", Size: tt.size, Digest: tt.digest}, path)
		if (err == nil) != tt.valid {
			t.Errorf("download %+v: %v", tt, err)
		}
		if !tt.valid {
			if _, err := os.Stat(path); err == nil {
				t.Error("corrupt download left available")
			}
		}
	}
}

func TestReleaseReadyUsesLatestUpdateAndStrictFiveMinuteBoundary(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name string
		age  time.Duration
		want bool
	}{
		{"刚发布", 0, false}, {"不到五分钟", 5*time.Minute - time.Nanosecond, false},
		{"恰好五分钟", 5 * time.Minute, false}, {"超过五分钟", 5*time.Minute + time.Nanosecond, true},
		{"未来时间", -time.Minute, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &Release{updatedAt: now.Add(-tt.age)}
			if got := r.Ready(now); got != tt.want {
				t.Fatalf("Ready = %v, want %v", got, tt.want)
			}
		})
	}
	if (&Release{}).Ready(now) {
		t.Fatal("缺失时间不能认可")
	}
	for _, field := range []string{"updated_at", "published_at", "asset"} {
		t.Run(field, func(t *testing.T) {
			data := strings.Replace(releaseJSON, `"body":`, `"published_at":"2020-01-01T00:00:00Z","body":`, 1)
			stamp := now.Add(-time.Minute).Format(time.RFC3339)
			if field == "asset" {
				data = strings.Replace(data, `"size":3`, `"updated_at":"`+stamp+`","size":3`, 1)
			} else {
				data = strings.Replace(data, `"`+field+`":"2020-01-01T00:00:00Z"`, `"`+field+`":"`+stamp+`"`, 1)
			}
			r, err := parseRelease([]byte(data), "darwin", "arm64")
			if err != nil {
				t.Fatal(err)
			}
			if r.Ready(now) {
				t.Fatal("最后一次修改不足五分钟不能认可")
			}
			if !r.Ready(now.Add(5 * time.Minute)) {
				t.Fatal("等待后应认可")
			}
		})
	}
}

type singleByteReader struct{ remaining []byte }

func (r *singleByteReader) Read(p []byte) (int, error) {
	if len(r.remaining) == 0 {
		return 0, io.EOF
	}
	p[0] = r.remaining[0]
	r.remaining = r.remaining[1:]
	return 1, nil
}

func TestDownloadProgressFollowsReceivedBytesAndStopsBeforeInstallation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		payload  string
		size     int64
		digest   string
		complete bool
	}{
		{"完整下载", "abcd", 4, "", true},
		{"大小不符", "abcde", 4, "", false},
		{"摘要不符", "abcd", 4, fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("wrong"))), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient()
			c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(&singleByteReader{remaining: []byte(tt.payload)}), Header: make(http.Header), Request: r}, nil
			})
			var progress []int
			c.Progress = func(phase, message string, percent int) {
				if phase != "downloading" {
					t.Errorf("phase = %s", phase)
				}
				progress = append(progress, percent)
			}
			path := t.TempDir() + "/asset"
			err := c.download(t.Context(), asset{Name: "asset", URL: "https://github.com/ixugo/lark-studio/releases/download/v1.0.0/asset", Size: tt.size, Digest: tt.digest}, path)
			if (err == nil) != tt.complete {
				t.Fatalf("download err = %v", err)
			}
			if len(progress) < 3 || progress[0] != 0 {
				t.Fatalf("missing byte progress: %v", progress)
			}
			for i, p := range progress {
				if p < 0 || p > 99 || (i > 0 && p < progress[i-1]) {
					t.Fatalf("invalid progress: %v", progress)
				}
			}
			if tt.complete && !slices.Equal(progress, []int{0, 24, 49, 74, 98, 99}) {
				t.Fatalf("分块进度不是按接收字节变化: %v", progress)
			}
			if tt.complete && progress[len(progress)-1] != 99 {
				t.Fatalf("completed download = %v", progress)
			}
			if !tt.complete && progress[len(progress)-1] == 99 {
				t.Fatalf("corrupt download completed: %v", progress)
			}
		})
	}
}
