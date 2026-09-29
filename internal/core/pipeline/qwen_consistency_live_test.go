package pipeline

import (
	"fmt"
	ttsadapter "github.com/ixugo/vdub/internal/adapter/tts"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// 受控多句文本与画面只用于验声，不冒充用户故障视频。
func TestLiveQwenVideoUsesOneVoice(t *testing.T) {
	if os.Getenv("LARK_QWEN_LIVE") != "1" {
		t.Skip("显式启用后在本机 Qwen 服务验声")
	}
	dir, err := filepath.Abs("../../../tmp/remote-voice-capabilities/consistency-video-final")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "source.mp4")
	command := exec.Command("ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=0x18232d:s=640x360:r=5", "-f", "lavfi", "-i", "anullsrc=r=24000:cl=mono", "-t", "120", "-c:v", "libx264", "-preset", "ultrafast", "-c:a", "aac", "-shortest", input)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("生成受控画面: %v %s", err, output)
	}
	texts := []string{
		"今天我们一起了解这个工具，看看怎样把一段视频翻译成另一种语言。",
		"先打开需要处理的视频，检查源语言，然后选择你需要的目标语言。",
		"小朋友问这个按钮该怎么使用，老师回答先把模型和声音选择好。",
		"一位男士说他已经完成这一步，另一位女士说她也准备好继续了。",
		"无论这句话提到谁，旁白都应该保持同一个人的声音和说话风格。",
		"请注意检查字幕内容，不要因为一句话比较短，就突然换成童声。",
		"现在我们进入下一步，等待系统生成每一句对应的配音和字幕。",
		"如果处理途中暂停，恢复任务后也应继续使用之前选定的声音。",
		"如果用户修改了配音音色，旧的音频缓存就不能与新配音混在一起。",
		"最后查看输出视频，确认文字、声音和时间轴都符合预期要求。",
		"感谢你耐心观看这段说明，希望这些步骤能够帮助你完成自己的作品。",
		"本段验证到这里结束，所有句子都使用同一套固定音色配置进行合成。",
	}
	content := ""
	for i, text := range texts {
		content += fmt.Sprintf("%d\n%s --> %s\n%s\n\n", i+1, fmt.Sprintf("00:%02d:%02d,000", i*10/60, i*10%60), fmt.Sprintf("00:%02d:%02d,000", (i*10+9)/60, (i*10+9)%60), text)
	}
	if err := os.WriteFile(filepath.Join(dir, "src.srt"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	router := ttsadapter.NewRouter("openai", "Vivian", "http://127.0.0.1:8399/v1", "not-needed", "mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-8bit")
	router.SetTTSConfig("openai", "Vivian", "http://127.0.0.1:8399/v1", "not-needed", "mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-8bit", ttsadapter.SpeechOptions{Protocol: "mlx", Language: "Auto"})
	core := NewCore(Config{FFmpegBin: "ffmpeg", MaxSpeedFactor: 1.2, SubtitleOutput: "none"}, nil, nil, router)
	job := Job{TaskID: "controlled-qwen-consistency", InputPath: input, OutputDir: dir, Mode: ModeDirectDub, TTSEngine: "openai", TTSVoice: "Vivian", SourceLang: "zh", TargetLang: "en", SpeechRate: 1, SubtitleOutput: "none"}
	if err := core.runTTS(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "audio_segs", "*.wav"))
	if err != nil || len(files) != len(texts) {
		t.Fatalf("配音没有生成全部句子: %d %v", len(files), err)
	}
	t.Logf("固定 Vivian / 自动按原文 Chinese：%d 句", len(files))
	if err := core.runMerge(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if err := core.runBurn(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	// 控制音色用相同文本，供独立说话人特征检查判断能否区分男、女声。
	for _, voice := range []string{"Ryan", "Serena"} {
		output := filepath.Join(dir, "control-"+voice+".wav")
		client := ttsadapter.NewOpenAITTS("http://127.0.0.1:8399/v1", "not-needed", "mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-8bit", voice).WithSpeechOptions(ttsadapter.SpeechOptions{Protocol: "mlx", Language: "Chinese"})
		if err := client.Synthesize(t.Context(), texts[0], output, voice); err != nil {
			t.Fatal(err)
		}
	}
}
