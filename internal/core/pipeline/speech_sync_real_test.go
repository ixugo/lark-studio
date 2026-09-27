//go:build integration

package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/ixugo/vdub/internal/adapter/llm"
	"github.com/ixugo/vdub/internal/adapter/tts"
	"github.com/ixugo/vdub/internal/adapter/whisper"
	"github.com/ixugo/vdub/internal/conf"
	"github.com/pelletier/go-toml/v2"
)

// 用户验收素材为 60000/1001 fps，按解码帧号比较可避开 GOP 寻址差异。
const speechSyncFrameRate = 60000.0 / 1001.0

// TestSpeechSync30sPipeline 用真实服务覆盖全部处理步骤，阻止音乐被误当作说话起点。
func TestSpeechSync30sPipeline(t *testing.T) {
	input, configPath := os.Getenv("VDUB_SYNC_VIDEO"), os.Getenv("VDUB_SYNC_CONFIG")
	if input == "" || configPath == "" {
		t.Skip("需要 VDUB_SYNC_VIDEO 和 VDUB_SYNC_CONFIG")
	}
	dir := os.Getenv("VDUB_SYNC_OUTPUT")
	if dir == "" {
		t.Fatal("VDUB_SYNC_OUTPUT 必须指定一个不存在的产物目录")
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := readSpeechSyncConfig(t, configPath)
	video := filepath.Join(dir, "source-30s.mp4")
	runSpeechSyncCommand(t, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", input,
		"-t", "30", "-c:v", "libx264", "-preset", "veryfast", "-crf", "18", "-c:a", "aac", video)
	notifier := &speechSyncRecorder{dir: dir}
	core := NewCore(Config{FFmpegBin: "ffmpeg", WhisperBin: cfg.Pipeline.WhisperBin,
		WhisperModel: cfg.Pipeline.WhisperModel, MaxSpeedFactor: 1.2, TTSWorkers: 2},
		whisper.NewRunner(cfg.Pipeline.WhisperBin, cfg.Pipeline.WhisperModel),
		llm.NewClient(cfg.LLM.BaseURL, cfg.LLM.APIKey, cfg.LLM.Model),
		tts.NewEdgeTTS(cfg.TTS.Voice), WithNotifier(notifier))
	job := Job{TaskID: "speech-sync-30s", InputPath: video, OutputDir: dir, Mode: ModeDub,
		SourceLang: "en", TargetLang: "zh-CN", Translator: "openai", TTSVoice: cfg.TTS.Voice,
		SubtitleOutput: "burn", OutputContent: "translated"}
	if err := core.Run(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if notifier.err != nil {
		t.Fatal(notifier.err)
	}
	want := []string{StepWhisper, StepSplit, StepTranslate, StepTTS, StepMerge, StepBurn}
	if !slices.Equal(notifier.steps, want) {
		t.Fatalf("实际步骤 %v，期望 %v", notifier.steps, want)
	}
	assertSpeechSyncSubtitles(t, dir)
	assertSpeechSyncAudio(t, dir)
	assertSpeechSyncFrames(t, dir)
	t.Logf("真实全流程成片：%s", filepath.Join(dir, "source-30s.final.mp4"))
}

// TestSpeechSyncDubPipeline 用真实识别、真实配音和 FFmpeg 验证音乐前奏不会触发片头配音。
func TestSpeechSyncDubPipeline(t *testing.T) {
	input, configPath := os.Getenv("VDUB_SYNC_VIDEO"), os.Getenv("VDUB_SYNC_CONFIG")
	if input == "" || configPath == "" {
		t.Skip("需要 VDUB_SYNC_VIDEO 和 VDUB_SYNC_CONFIG")
	}
	cfg := readSpeechSyncConfig(t, configPath)
	dir := t.TempDir()
	notifier := &speechSyncRecorder{dir: dir}
	core := NewCore(Config{
		FFmpegBin: "ffmpeg", WhisperBin: cfg.Pipeline.WhisperBin,
		WhisperModel: cfg.Pipeline.WhisperModel, MaxSpeedFactor: 1.2, TTSWorkers: 2,
	}, whisper.NewRunner(cfg.Pipeline.WhisperBin, cfg.Pipeline.WhisperModel),
		speechSyncStubLLM{}, tts.NewEdgeTTS(cfg.TTS.Voice), WithNotifier(notifier))
	job := Job{
		TaskID: "speech-sync-dub", InputPath: input, OutputDir: dir,
		Mode: ModeDub, SourceLang: "en", TargetLang: "zh-CN", Translator: "openai",
		OutputContent: "translated", TTSEngine: "edge", TTSVoice: cfg.TTS.Voice,
		SubtitleOutput: "burn",
	}
	if err := core.Run(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	want := []string{StepWhisper, StepSplit, StepTranslate, StepTTS, StepMerge, StepBurn}
	if !slices.Equal(notifier.steps, want) {
		t.Fatalf("实际步骤 %v，期望 %v", notifier.steps, want)
	}
	entries := readSpeechSyncSRT(t, dir, "trans.srt")
	if entries[0].StartSec < 15.1 || entries[0].StartSec > 15.8 {
		t.Fatalf("首条配音字幕 %.3fs，期望落在人声起点", entries[0].StartSec)
	}
	assertSpeechSyncAudio(t, dir)
}

type speechSyncStubLLM struct{}

// SplitSentences 保持真实听写全文为单句，隔离语义服务对时间轴验收的影响。
func (speechSyncStubLLM) SplitSentences(_ context.Context, text, _ string) ([]string, error) {
	return []string{text}, nil
}

// Translate 返回固定中文句子，避免测试依赖外部翻译服务。
func (speechSyncStubLLM) Translate(
	_ context.Context,
	sentences []string,
	_, _ string,
	_, _ []string,
) ([]string, error) {
	translated := make([]string, len(sentences))
	for i := range translated {
		translated[i] = "大家好，我来帮助你设计理想生活。"
	}
	return translated, nil
}

// readSpeechSyncConfig 只读取实际服务配置，避免把密钥写入测试和日志。
func readSpeechSyncConfig(t *testing.T, path string) conf.Bootstrap {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg conf.Bootstrap
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// speechSyncRecorder 留下每一步的字幕快照，区分识别错误与后续时轴篡改。
type speechSyncRecorder struct {
	noopNotifier
	dir   string
	steps []string
	err   error
	mu    sync.Mutex
}

// OnStepDone 按执行顺序记录实际完成的步骤及字幕内容。
func (n *speechSyncRecorder) OnStepDone(taskID, step string) {
	n.steps = append(n.steps, step)
	for _, name := range []string{"src.srt", "trans.srt"} {
		data, err := os.ReadFile(filepath.Join(n.dir, name))
		if os.IsNotExist(err) {
			continue
		}
		if err == nil {
			err = os.WriteFile(filepath.Join(n.dir, step+"-"+name), data, 0o644)
		}
		if err != nil {
			n.err = err
		}
	}
}

// OnLog 串行保存实际引擎输出，使服务响应和语音区间可供复核。
func (n *speechSyncRecorder) OnLog(taskID, message string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	f, err := os.OpenFile(filepath.Join(n.dir, "engines.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		n.err = err
		return
	}
	if _, err := fmt.Fprintln(f, message); err != nil {
		n.err = err
	}
	if err := f.Close(); err != nil {
		n.err = err
	}
}

// readSpeechSyncSRT 将真实产物解析为断言输入，不生成理想时间戳替代识别。
func readSpeechSyncSRT(t *testing.T, dir, name string) []srtEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	entries := parseSRT(string(data))
	if len(entries) == 0 {
		t.Fatalf("%s 没有字幕", name)
	}
	return entries
}

// assertSpeechSyncSubtitles 检查前三句的原片位置、译文及各阶段时间轴不漂移。
func assertSpeechSyncSubtitles(t *testing.T, dir string) {
	t.Helper()
	raw := readSpeechSyncSRT(t, dir, "whisper-src.srt")
	if raw[0].StartSec < 15.1 || raw[0].StartSec > 15.8 {
		t.Errorf("听写首句 %.3fs，期望真实人声约15秒，不能取音乐起点", raw[0].StartSec)
	}
	source := readSpeechSyncSRT(t, dir, "split-src.srt")
	translated := readSpeechSyncSRT(t, dir, "trans.srt")
	if len(source) < 3 || len(source) != len(translated) {
		t.Fatalf("源/译文数量：%d/%d", len(source), len(translated))
	}
	for i, bounds := range [][2]float64{{15.1, 15.8}, {16.0, 17.1}, {18.1, 19.1}} {
		if source[i].StartSec < bounds[0] || source[i].StartSec > bounds[1] {
			t.Errorf("第%d句起点 %.3f 超出原片区间 %v", i+1, source[i].StartSec, bounds)
		}
		t.Logf("第%d句：%.3f–%.3f / %s / %s", i+1, source[i].StartSec, source[i].EndSec, source[i].Text, translated[i].Text)
	}
	for i, entry := range source {
		if entry.StartSec != translated[i].StartSec || entry.EndSec != translated[i].EndSec {
			t.Errorf("第%d句翻译时轴变化", i+1)
		}
		if entry.EndSec <= entry.StartSec || (i > 0 && entry.StartSec < source[i-1].EndSec) {
			t.Errorf("第%d句时轴倒退或重叠", i+1)
		}
		if !hasCJK([]rune(translated[i].Text)) {
			t.Errorf("第%d句没有中文译文", i+1)
		}
	}
	for _, name := range []string{"src.srt", "trans.srt"} {
		before, err := os.ReadFile(filepath.Join(dir, "tts-"+name))
		if err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(filepath.Join(dir, "merge-"+name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("混音篡改了 %s", name)
		}
	}
}

// runSpeechSyncCommand 检查外部工具的退出状态，保留失败输出。
func runSpeechSyncCommand(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v\n%s", name, err, output)
	}
	return output
}

// assertSpeechSyncAudio 验证整段片头无配音，三句配音均存在且最终成片仍保留音乐。
func assertSpeechSyncAudio(t *testing.T, dir string) {
	t.Helper()
	entries := readSpeechSyncSRT(t, dir, "trans.srt")
	dub := speechSyncPCM(t, filepath.Join(dir, "dub.mp3"))
	if peakSpeechSyncPCM(dub, 0, 15.0) > 0.002 {
		t.Error("配音在原片人声之前抢跑")
	}
	for i, entry := range entries[:min(3, len(entries))] {
		if peakSpeechSyncPCM(dub, entry.StartSec, entry.EndSec) < 0.01 {
			t.Errorf("第%d句缺少真实配音", i+1)
		}
	}
	final := filepath.Join(dir, "source-30s.final.mp4")
	pcm := speechSyncPCM(t, final)
	for i, entry := range entries[:min(3, len(entries))] {
		correlation := speechSyncCorrelation(dub, pcm, entry.StartSec, entry.EndSec)
		if correlation < 0.75 {
			t.Errorf("第%d句最终音轨与配音不对应：%.3f", i+1, correlation)
		}
		t.Logf("第%d句最终音轨配音相关系数 %.3f", i+1, correlation)
	}
	onset := -1.0
	for i := 0; i+1 < len(dub); i += 2 {
		if math.Abs(float64(int16(uint16(dub[i])|uint16(dub[i+1])<<8)))/32768 > 0.01 {
			onset = float64(i) / 32000
			break
		}
	}
	if onset < 15.1 || onset > 15.8 || math.Abs(onset-entries[0].StartSec) > 0.15 {
		t.Errorf("首句配音 %.3fs 与字幕 %.3fs 不同步", onset, entries[0].StartSec)
	}
	t.Logf("首句配音实际可闻起点 %.3fs", onset)
	if peakSpeechSyncPCM(pcm, 4.5, 10) < 0.002 {
		t.Error("成片丢失片头音乐")
	}
	if duration := probeMediaDuration("ffmpeg", final); math.Abs(duration-30) > 0.15 {
		t.Errorf("成片时长 %.3f", duration)
	}
}

// speechSyncCorrelation 检查最终混合音轨确含同时间窗的配音，而非仅检测原声的音量。
func speechSyncCorrelation(left, right []byte, start, end float64) float64 {
	var cross, leftPower, rightPower float64
	for i := max(0, int(start*16000)) * 2; i+1 < min(len(left), len(right), int(end*16000)*2); i += 2 {
		a := float64(int16(uint16(left[i]) | uint16(left[i+1])<<8))
		b := float64(int16(uint16(right[i]) | uint16(right[i+1])<<8))
		cross += a * b
		leftPower += a * a
		rightPower += b * b
	}
	if leftPower == 0 || rightPower == 0 {
		return 0
	}
	return cross / math.Sqrt(leftPower*rightPower)
}

// speechSyncPCM 解码实际音轨供逐样本检查，复杂度随30秒音轨线性增长。
func speechSyncPCM(t *testing.T, path string) []byte {
	t.Helper()
	return runSpeechSyncCommand(t, "ffmpeg", "-v", "error", "-i", path, "-vn", "-ar", "16000", "-ac", "1", "-f", "s16le", "-")
}

// peakSpeechSyncPCM 遍历整个待验区间，避免仅凭一个静音起点判定通过。
func peakSpeechSyncPCM(data []byte, from, to float64) float64 {
	peak := 0.0
	for i := max(0, int(from*16000)) * 2; i+1 < min(len(data), int(to*16000)*2); i += 2 {
		value := int16(uint16(data[i]) | uint16(data[i+1])<<8)
		peak = max(peak, math.Abs(float64(value))/32768)
	}
	return peak
}

// assertSpeechSyncFrames 比较独立无字幕编码，要求片头无字幕且三句窗口出现字幕像素。
func assertSpeechSyncFrames(t *testing.T, dir string) {
	t.Helper()
	baseline := filepath.Join(dir, "picture-control.mp4")
	runSpeechSyncCommand(t, "ffmpeg", "-v", "error", "-y", "-i", filepath.Join(dir, "source-30s.mp4"), "-filter_complex", "[0:v]null[v]", "-map", "[v]", "-an", "-preset", "veryfast", baseline)
	entries := readSpeechSyncSRT(t, dir, "trans.srt")
	for _, sec := range []float64{5, 10, 14, 15.2, entries[0].StartSec + 0.3, entries[1].StartSec + 0.3, entries[2].StartSec + 0.3} {
		frame := speechSyncFrame(t, filepath.Join(dir, "source-30s.final.mp4"), sec)
		control := speechSyncFrame(t, baseline, sec)
		if len(frame) != len(control) {
			t.Fatal("画面对照尺寸不同")
		}
		changed := 0
		for i, pixel := range frame {
			if math.Abs(float64(pixel)-float64(control[i])) > 45 {
				changed++
			}
		}
		if sec < 15.3 && changed > 30 {
			t.Errorf("%.3fs 提前出现字幕像素：%d", sec, changed)
		}
		if sec > entries[0].StartSec && changed < 100 {
			t.Errorf("%.3fs 缺少字幕像素：%d", sec, changed)
		}
		t.Logf("画面 %.3fs：字幕差异像素 %d", sec, changed)
		runSpeechSyncCommand(t, "ffmpeg", "-v", "error", "-y", "-ss", fmt.Sprintf("%.3f", sec), "-i", filepath.Join(dir, "source-30s.final.mp4"), "-frames:v", "1", filepath.Join(dir, fmt.Sprintf("frame-%.3f.png", sec)))
	}
}

// TestSpeechSyncVariableIntro 用同一真实素材延长前奏和只取音乐，排除固定偏移与音乐误识别。
func TestSpeechSyncVariableIntro(t *testing.T) {
	input, configPath := os.Getenv("VDUB_SYNC_VIDEO"), os.Getenv("VDUB_SYNC_CONFIG")
	if input == "" || configPath == "" {
		t.Skip("需要 VDUB_SYNC_VIDEO 和 VDUB_SYNC_CONFIG")
	}
	cfg := readSpeechSyncConfig(t, configPath)
	runner := whisper.NewRunner(cfg.Pipeline.WhisperBin, cfg.Pipeline.WhisperModel)
	for _, test := range []struct {
		name     string
		duration string
		filter   string
		start    float64
	}{
		{"longer-intro", "30", "adelay=3000:all=1", 18.4},
		{"music-only", "10", "anull", -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			wav, srt := filepath.Join(dir, "input.wav"), filepath.Join(dir, "output.srt")
			runSpeechSyncCommand(t, "ffmpeg", "-v", "error", "-y", "-i", input, "-t", test.duration, "-vn", "-af", test.filter, "-ar", "16000", "-ac", "1", wav)
			if err := runner.Transcribe(t.Context(), wav, srt, "en", nil, nil); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(srt)
			if err != nil {
				t.Fatal(err)
			}
			entries := parseSRT(string(data))
			if test.start < 0 {
				if len(entries) != 0 {
					t.Fatalf("纯音乐被听写为人声：%s", data)
				}
				return
			}
			if len(entries) == 0 || math.Abs(entries[0].StartSec-test.start) > 0.4 {
				t.Fatalf("加长前奏未同步后移：%s", data)
			}
			t.Logf("前奏延长3秒，首句 %.3fs", entries[0].StartSec)
		})
	}
}

// speechSyncFrame 提取字幕区域灰度像素，使烧录结果可由自动断言复核。
func speechSyncFrame(t *testing.T, video string, sec float64) []byte {
	t.Helper()
	filter := fmt.Sprintf("select=eq(n\\,%d),crop=iw:ih/3:0:ih*2/3,scale=640:-1", int(sec*speechSyncFrameRate))
	return runSpeechSyncCommand(t, "ffmpeg", "-v", "error", "-i", video, "-frames:v", "1", "-vf", filter, "-pix_fmt", "gray", "-f", "rawvideo", "-")
}
