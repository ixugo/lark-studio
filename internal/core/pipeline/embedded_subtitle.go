package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const subtitleProbeTimeout = 30 * time.Second

// 只提取文字轨；位图字幕需要 OCR，仍交给语音识别。
var subtitleStreamPattern = regexp.MustCompile(`Stream #0:(\d+)(?:\[[^\]]+\])?(?:\(([^)]+)\))?: Subtitle: (\w+)([^\n]*)`)

type subtitleTrack struct {
	index     int
	language  string
	preferred bool
}
type RecognitionMedia struct{ FFmpeg, SourceLang string }

func subtitleLanguage(value string) string {
	value = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	parts := strings.SplitN(value, "-", 2)
	primary := parts[0]
	switch primary {
	case "eng":
		primary = "en"
	case "zho", "chi":
		primary = "zh"
	case "jpn":
		primary = "ja"
	case "kor":
		primary = "ko"
	case "fra", "fre":
		primary = "fr"
	case "deu", "ger":
		primary = "de"
	case "spa":
		primary = "es"
	case "rus":
		primary = "ru"
	case "ita":
		primary = "it"
	case "por":
		primary = "pt"
	}
	if len(parts) == 2 {
		return primary + "-" + parts[1]
	}
	return primary
}
func selectSubtitleTrack(text, language string) (subtitleTrack, bool) {
	var tracks []subtitleTrack
	for _, match := range subtitleStreamPattern.FindAllStringSubmatch(text, -1) {
		switch match[3] {
		case "subrip", "srt", "ass", "ssa", "mov_text", "webvtt", "text":
		default:
			continue
		}
		if strings.Contains(match[4], "(forced)") {
			continue
		}
		index, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		tracks = append(tracks, subtitleTrack{index: index, language: subtitleLanguage(match[2]), preferred: strings.Contains(match[4], "(default)")})
	}
	target := subtitleLanguage(language)
	if target != "" && target != "auto" {
		for _, track := range tracks {
			if track.language == target {
				return track, true
			}
		}
		var family []subtitleTrack
		for _, track := range tracks {
			if strings.Split(track.language, "-")[0] == strings.Split(target, "-")[0] {
				family = append(family, track)
			}
		}
		for _, track := range family {
			if track.preferred {
				return track, true
			}
		}
		if len(family) == 1 {
			return family[0], true
		}
		return subtitleTrack{}, false
	}
	for _, track := range tracks {
		if track.preferred {
			return track, true
		}
	}
	if len(tracks) == 1 {
		return tracks[0], true
	}
	return subtitleTrack{}, false
}

// ExtractEmbeddedSubtitle 仅在得到可解析的 SRT 后发布文件，失败不覆盖已有字幕。
func ExtractEmbeddedSubtitle(ctx context.Context, ffmpeg, input, language, output string) (bool, error) {
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}
	probeCtx, cancel := context.WithTimeout(ctx, subtitleProbeTimeout)
	defer cancel()
	details, err := exec.CommandContext(probeCtx, ffmpeg, "-nostdin", "-hide_banner", "-i", input).CombinedOutput()
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if probeCtx.Err() != nil {
		return false, probeCtx.Err()
	}
	// 不指定输出文件时 FFmpeg 会以非零退出；流信息仍在标准错误中。
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return false, err
	}
	track, ok := selectSubtitleTrack(string(details), language)
	if !ok {
		return false, nil
	}
	file, err := os.CreateTemp(filepath.Dir(output), ".subtitle-*.srt")
	if err != nil {
		return false, err
	}
	temp := file.Name()
	if err := file.Close(); err != nil {
		return false, err
	}
	defer func() {
		if err := os.Remove(temp); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("清理字幕临时文件失败", "err", err)
		}
	}()
	data, err := exec.CommandContext(probeCtx, ffmpeg, "-nostdin", "-v", "error", "-y", "-i", input, "-map", fmt.Sprintf("0:%d", track.index), "-c:s", "srt", "-f", "srt", temp).CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		return false, fmt.Errorf("提取内嵌字幕失败: %w (%s)", err, strings.TrimSpace(string(data)))
	}
	data, err = os.ReadFile(temp)
	if err != nil {
		return false, err
	}
	if len(parseSRT(string(data))) == 0 {
		return false, nil
	}
	if err := os.Rename(temp, output); err != nil {
		return false, err
	}
	return true, nil
}

func hasEmbeddedRecognitionSubtitle(input string, media RecognitionMedia) bool {
	dir, err := os.MkdirTemp("", "lark-subtitle-check-*")
	if err != nil {
		return false
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.Warn("清理字幕预检目录失败", "err", err)
		}
	}()
	ok, err := ExtractEmbeddedSubtitle(context.Background(), media.FFmpeg, input, media.SourceLang, filepath.Join(dir, "src.srt"))
	return err == nil && ok
}
