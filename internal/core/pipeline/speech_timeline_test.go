package pipeline

import (
	"encoding/binary"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestNormalizeSpeechKeepsOnlyInternalSilence 防止 TTS 自带片头静音拖晚配音，同时保留句中停顿。
func TestNormalizeSpeechKeepsOnlyInternalSilence(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("需要 ffmpeg")
	}
	path := filepath.Join(t.TempDir(), "0.wav")
	cmd := exec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i",
		"aevalsrc=if(between(t\\,0.2\\,0.5)+between(t\\,0.8\\,1.1)\\,0.2*sin(2*PI*440*t)\\,0):s=24000:d=1.4", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("生成音频：%v %s", err, out)
	}
	if _, err := normalizeAudioSegment(t.Context(), "ffmpeg", path); err != nil {
		t.Fatal(err)
	}
	pcm, err := exec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-i", path, "-f", "s16le", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	if testSpeechPeak(pcm, 0, 0.08) < 0.1 {
		t.Error("配音仍有片头静音，无法贴合字幕起点")
	}
	if testSpeechPeak(pcm, 0.4, 0.5) > 0.002 {
		t.Error("句中停顿被删除")
	}
	if testSpeechPeak(pcm, 0.7, 0.8) < 0.1 {
		t.Error("后半句音频丢失")
	}
}

// testSpeechPeak 检查整个 PCM 时间窗，避免总时长正确掩盖内容丢失。
func testSpeechPeak(pcm []byte, start, end float64) float64 {
	peak := 0.0
	for i := int(start*24000) * 2; i+1 < min(len(pcm), int(end*24000)*2); i += 2 {
		peak = max(peak, math.Abs(float64(int16(binary.LittleEndian.Uint16(pcm[i:]))))/32768)
	}
	return peak
}

// TestMergeNeverRewritesSpeechTimeline 验证开头立即说话时，混音不会探测后把后续句整体平移。
func TestMergeNeverRewritesSpeechTimeline(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("需要 ffmpeg")
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "audio_segs"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "audio_segs", "0.wav")
	cmd := exec.CommandContext(t.Context(), "ffmpeg", "-v", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=440:duration=0.5", "-ar", "24000", "-ac", "1", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("生成配音：%v %s", err, out)
	}
	srt := "1\n00:00:00,000 --> 00:00:00,500\n大家好\n\n"
	if err := os.WriteFile(filepath.Join(dir, "trans.srt"), []byte(srt), 0o644); err != nil {
		t.Fatal(err)
	}
	core := NewCore(Config{FFmpegBin: "ffmpeg"}, nil, nil, nil)
	if err := core.runMerge(t.Context(), Job{OutputDir: dir, InputPath: "unused.mp4"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "trans.srt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != srt {
		t.Fatalf("混音改写了听写时间轴：%s", got)
	}
}
