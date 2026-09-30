package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	latestURL       = "https://api.github.com/repos/ixugo/lark-studio/releases/latest"
	maxReleaseBytes = 1 << 20
	maxAssetBytes   = 1 << 30
)

type Client struct {
	HTTP     *http.Client
	Progress func(phase, message string, percent int)
}

type asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
	State  string `json:"state"`
}

type Release struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	asset   asset
	goos    string
	goarch  string
	reason  string
}

func (r *Release) Supported() bool { return r != nil && r.reason == "" && r.asset.Name != "" }
func (r *Release) Reason() string {
	if r == nil {
		return "尚未检查更新"
	}
	return r.reason
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 20 * time.Minute, CheckRedirect: trustedRedirect}}
}

func trustedRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("下载重定向过多")
	}
	u := req.URL
	if u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return errors.New("更新下载重定向不安全")
	}
	switch u.Hostname() {
	case "github.com", "api.github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return nil
	default:
		return errors.New("更新下载重定向至非 GitHub 地址")
	}
}

func (c *Client) Fetch(ctx context.Context, goos, goarch string) (*Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	response, err := c.request(ctx, latestURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxReleaseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取更新信息: %w", err)
	}
	if len(data) > maxReleaseBytes {
		return nil, errors.New("更新信息超过大小限制")
	}
	return parseRelease(data, goos, goarch)
}

func parseRelease(data []byte, goos, goarch string) (*Release, error) {
	var body struct {
		Tag        string  `json:"tag_name"`
		Body       string  `json:"body"`
		Draft      bool    `json:"draft"`
		Prerelease bool    `json:"prerelease"`
		Assets     []asset `json:"assets"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("更新信息格式错误: %w", err)
	}
	if _, err := versionParts(body.Tag); err != nil {
		return nil, err
	}
	if body.Draft || body.Prerelease {
		return nil, errors.New("更新信息不是正式发行版")
	}
	r := &Release{Version: body.Tag, Notes: body.Body, goos: goos, goarch: goarch}
	suffix := ""
	switch goos + "/" + goarch {
	case "darwin/arm64":
		suffix = "_macos_arm64.dmg"
	case "windows/amd64":
		suffix = "_windows_amd64.zip"
	}
	if suffix == "" {
		r.reason = "当前系统或架构不支持自动安装"
		return r, nil
	}
	pattern := regexp.MustCompile(`^lark-studio_v?[0-9]+\.[0-9]+\.[0-9]+` + regexp.QuoteMeta(suffix) + `$`)
	for _, a := range body.Assets {
		if a.State != "uploaded" || !pattern.MatchString(a.Name) {
			continue
		}
		if r.asset.Name != "" {
			return nil, errors.New("发行版包含多个匹配的安装包")
		}
		if err := validateAsset(a, body.Tag); err != nil {
			return nil, err
		}
		r.asset = a
	}
	if r.asset.Name == "" {
		r.reason = "最新版本的当前系统安装包尚未上传完成，请稍后再检查"
	}
	return r, nil
}

func validateAsset(a asset, tag string) error {
	if a.Size <= 0 || a.Size > maxAssetBytes {
		return errors.New("更新安装包大小无效")
	}
	u, err := url.Parse(a.URL)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("更新安装包地址不受信任")
	}
	if u.Path != "/ixugo/lark-studio/releases/download/"+tag+"/"+a.Name {
		return errors.New("更新安装包地址与发行版不符")
	}
	if a.Digest != "" {
		if !strings.HasPrefix(a.Digest, "sha256:") {
			return errors.New("更新安装包摘要格式无效")
		}
		decoded, err := hex.DecodeString(strings.TrimPrefix(a.Digest, "sha256:"))
		if err != nil || len(decoded) != sha256.Size {
			return errors.New("更新安装包摘要格式无效")
		}
	}
	return nil
}

func (c *Client) request(ctx context.Context, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Lark-Studio-Updater")
	client := c.HTTP
	if client == nil {
		client = NewClient().HTTP
	}
	// 即使调用方设置传输层，正式下载也必须保留地址校验。
	copyClient := *client
	copyClient.CheckRedirect = trustedRedirect
	resp, err := copyClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("访问 GitHub 更新服务: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		closeErr := resp.Body.Close()
		return nil, errors.Join(fmt.Errorf("GitHub 更新服务返回 HTTP %d", resp.StatusCode), closeErr)
	}
	return resp, nil
}

func (c *Client) download(ctx context.Context, a asset, path string) (err error) {
	resp, err := c.request(ctx, a.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(path))
		}
	}()
	hash := sha256.New()
	c.progress("downloading", "正在下载更新安装包", 0)
	progress := &downloadProgressWriter{writer: io.MultiWriter(f, hash), total: a.Size, report: func(percent int) {
		c.progress("downloading", "正在下载更新安装包", percent)
	}}
	n, copyErr := io.Copy(progress, io.LimitReader(resp.Body, a.Size+1))
	err = errors.Join(copyErr, f.Sync(), f.Close())
	if err != nil {
		return err
	}
	if n != a.Size {
		return fmt.Errorf("更新安装包大小不符: 收到 %d，预期 %d", n, a.Size)
	}
	if a.Digest != "" && !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), strings.TrimPrefix(a.Digest, "sha256:")) {
		return errors.New("更新安装包 SHA256 校验失败")
	}
	c.progress("downloading", "正在下载更新安装包", 99)
	return nil
}

type downloadProgressWriter struct {
	writer      io.Writer
	total       int64
	written     int64
	lastPercent int
	report      func(int)
}

func (w *downloadProgressWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	w.written += int64(n)
	percent := min(98, int(w.written*99/w.total))
	if percent > w.lastPercent {
		w.lastPercent = percent
		w.report(percent)
	}
	return n, err
}

func (c *Client) progress(phase, message string, percent int) {
	if c.Progress != nil {
		c.Progress(phase, message, percent)
	}
}
