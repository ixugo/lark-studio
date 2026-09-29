package update

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
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

const releaseJSON = `{"tag_name":"v0.1.0","body":"更新说明","assets":[{"name":"lark-studio_v0.0.127_macos_arm64.dmg","size":3,"browser_download_url":"https://github.com/ixugo/lark-studio/releases/download/v0.1.0/lark-studio_v0.0.127_macos_arm64.dmg"}]}`

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
