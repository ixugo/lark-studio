package pipeline

import (
	"context"
	"errors"
	"github.com/ixugo/vdub/internal/adapter/asr"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEmbeddedSubtitleReuse(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("需要 ffmpeg")
	}
	for _, tc := range []struct{ codec, extension string }{{"mov_text", ".mp4"}, {"srt", ".mkv"}, {"ass", ".mkv"}} {
		t.Run(tc.codec, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "captions.srt")
			if err := os.WriteFile(source, []byte("1\n00:00:00,000 --> 00:00:00,500\nActual subtitle\n\n"), 0600); err != nil {
				t.Fatal(err)
			}
			video := filepath.Join(dir, "video"+tc.extension)
			output, err := exec.CommandContext(t.Context(), ffmpeg, "-v", "error", "-f", "lavfi", "-i", "color=size=32x32:duration=1", "-i", source, "-c:v", "mpeg4", "-c:s", tc.codec, "-metadata:s:s:0", "language=eng", video).CombinedOutput()
			if err != nil {
				t.Fatalf("%v: %s", err, output)
			}
			extracted := filepath.Join(dir, "result.srt")
			ok, err := ExtractEmbeddedSubtitle(t.Context(), ffmpeg, video, "en", extracted)
			if err != nil || !ok {
				t.Fatalf("ok=%v err=%v", ok, err)
			}
			data, err := os.ReadFile(extracted)
			if err != nil || len(parseSRT(string(data))) != 1 {
				t.Fatalf("subtitle=%s err=%v", data, err)
			}
			if err := ValidateRecognitionConfig(video, "", ModeSubtitle, asr.Config{}, RecognitionMedia{FFmpeg: ffmpeg, SourceLang: "en"}); err != nil {
				t.Fatal(err)
			}
			ok, err = ExtractEmbeddedSubtitle(t.Context(), ffmpeg, video, "zh", filepath.Join(dir, "wrong.srt"))
			if err != nil || ok {
				t.Fatalf("wrong language ok=%v err=%v", ok, err)
			}
			core := Core{cfg: Config{FFmpegBin: ffmpeg}, notifier: &noopNotifier{}}
			if err := core.runWhisper(t.Context(), Job{InputPath: video, OutputDir: dir, SourceLang: "en"}); err != nil {
				t.Fatal(err)
			}
			// 未装配识别器；如字幕没有复用，本测试会在听写调用处失败。
		})
	}
}

func TestSubtitleTrackSelection(t *testing.T) {
	text := "Stream #0:2(eng): Subtitle: subrip (default)\nStream #0:3(zho): Subtitle: mov_text\nStream #0:4(eng): Subtitle: subrip (forced)\nStream #0:5(jpn): Subtitle: hdmv_pgs_subtitle\n"
	for _, tc := range []struct {
		language string
		index    int
		ok       bool
	}{{"auto", 2, true}, {"en", 2, true}, {"zh-CN", 3, true}, {"ja", 0, false}, {"fr", 0, false}} {
		track, ok := selectSubtitleTrack(text, tc.language)
		if ok != tc.ok || (ok && track.index != tc.index) {
			t.Fatalf("%s: %+v %v", tc.language, track, ok)
		}
	}
	if _, ok := selectSubtitleTrack("Stream #0:2(eng): Subtitle: subrip\nStream #0:3(zho): Subtitle: subrip", "auto"); ok {
		t.Fatal("多语言无默认轨不能任意选取")
	}
}

func TestSubtitleRegionalLanguage(t *testing.T) {
	text := "Stream #0:2(pt-PT): Subtitle: subrip (default)\nStream #0:3(pt-BR): Subtitle: subrip\n"
	track, ok := selectSubtitleTrack(text, "pt-BR")
	if !ok || track.index != 3 {
		t.Fatalf("%+v %v", track, ok)
	}
}

type subtitleFallbackWhisper struct{ calls int }

func (w *subtitleFallbackWhisper) Transcribe(ctx context.Context, input, output, language string, progress func(int), log func(string)) error {
	w.calls++
	return os.WriteFile(output, []byte("1\n00:00:00,000 --> 00:00:00,100\nRecognized\n\n"), 0600)
}
func TestMissingSubtitleFallsBackToRecognition(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("需要 ffmpeg")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "audio.wav")
	output, err := exec.CommandContext(t.Context(), ffmpeg, "-v", "error", "-f", "lavfi", "-i", "sine=duration=0.2", input).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, output)
	}
	whisper := &subtitleFallbackWhisper{}
	core := Core{cfg: Config{FFmpegBin: ffmpeg}, notifier: &noopNotifier{}, whisper: whisper}
	if err := core.runWhisper(t.Context(), Job{InputPath: input, OutputDir: dir}); err != nil {
		t.Fatal(err)
	}
	if whisper.calls != 1 {
		t.Fatalf("听写调用次数=%d", whisper.calls)
	}
	if err := ValidateRecognitionConfig(input, "", ModeSubtitle, asr.Config{}, RecognitionMedia{FFmpeg: ffmpeg}); err == nil {
		t.Fatal("无字幕也无识别引擎应拒绝")
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if ok, err := ExtractEmbeddedSubtitle(cancelled, ffmpeg, input, "auto", filepath.Join(dir, "cancel.srt")); ok || !errors.Is(err, context.Canceled) {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
