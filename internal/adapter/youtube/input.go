package youtube

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const maxURLLength = 2048

var videoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func NormalizeURL(raw string) (string, error) {
	if len(raw) > maxURLLength {
		return "", errors.New("链接过长")
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return "", errors.New("请输入有效的 HTTPS YouTube 视频链接")
	}
	var id string
	switch strings.ToLower(u.Host) {
	case "youtu.be":
		id = strings.TrimPrefix(u.Path, "/")
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else {
			parts := strings.Split(strings.Trim(u.Path, "/"), "/")
			if len(parts) == 2 && slices.Contains([]string{"shorts", "embed", "live"}, parts[0]) {
				id = parts[1]
			}
		}
	}
	if !videoID.MatchString(id) {
		return "", errors.New("仅支持单个 YouTube 视频链接，不支持播放列表")
	}
	return "https://www.youtube.com/watch?v=" + id, nil
}

func DownloadDirectory(outputDir string) (string, error) {
	outputDir = strings.TrimSpace(outputDir)
	if outputDir == "" {
		return "", errors.New("请先在设置中配置默认输出目录")
	}
	if outputDir == "~" || strings.HasPrefix(outputDir, "~/") || strings.HasPrefix(outputDir, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		outputDir = filepath.Join(home, strings.TrimLeft(strings.TrimPrefix(outputDir, "~"), "/\\"))
	}
	absolute, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(absolute, "Downloads"), nil
}
