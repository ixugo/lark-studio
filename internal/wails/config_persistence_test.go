package wails

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ixugo/vdub/internal/conf"
)

// 模拟页面保存、重新读取和应用重启，按浏览器实际使用的小写字段核对全部输入。
func TestSettingsFormsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	asrServer := newRemoteModelTestServer(t, "custom-asr")
	translationServer := newRemoteModelTestServer(t, "custom-translate")
	ttsServer := newTTSConfigurationFixture(t, "custom-tts")
	bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}}
	svc := &AppService{bc: &bc}
	updates := map[string]any{
		"pipeline": map[string]any{"workers": 3, "whisper_mode": "openai", "whisper_bin": "/test/whisper-cli", "whisper_model": "/test/ggml.bin", "asr_base_url": asrServer.URL + "/v1", "asr_api_key": "fixture-asr", "asr_model": "custom-asr", "ffmpeg_bin": "/test/ffmpeg", "default_output_dir": "/test/output", "default_target_lang": "ja", "translate_prompt": "保持术语", "translate_chunk_size": 7, "max_speed_factor": 1.35, "tts_workers": 3, "clean_intermediate": true, "subtitle_output": "none"},
		"llm":      map[string]any{"provider": "openai", "base_url": translationServer.URL + "/v1", "api_key": "fixture-translate", "model": "custom-translate", "deeplx_url": "https://deeplx.example"},
		"tts":      map[string]any{"type": "openai", "voice": "custom-voice", "edge_voice": "en-US-JennyNeural", "openai_voice": "custom-voice", "base_url": ttsServer.URL + "/v1", "api_key": "fixture-tts", "model": "custom-tts"},
		"lip_sync": map[string]any{"enabled": true, "base_url": "https://lipsync.example", "api_key": "fixture-lipsync"},
	}
	raw, _ := json.Marshal(updates)
	var browserUpdates map[string]any
	if err := json.Unmarshal(raw, &browserUpdates); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpdateConfig(browserUpdates); err != nil {
		t.Fatal(err)
	}
	check := func(label string, service *AppService) {
		t.Helper()
		raw, err := json.Marshal(service.GetConfig())
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		for section, expected := range browserUpdates {
			actual, ok := got[section].(map[string]any)
			if !ok {
				t.Errorf("%s 缺少配置区域 %s", label, section)
				continue
			}
			for field, value := range expected.(map[string]any) {
				if !reflect.DeepEqual(actual[field], value) {
					t.Errorf("%s %s.%s 保存后=%v，期望=%v", label, section, field, actual[field], value)
				}
			}
		}
	}
	check("菜单切换", svc)
	var loaded conf.Bootstrap
	if err := conf.SetupConfig(&loaded, path); err != nil {
		t.Fatal(err)
	}
	loaded.Runtime.ConfigPath = path
	check("重新启动", &AppService{bc: &loaded})
}

func TestSettingsSaveFailureLeavesAllSectionsUnchanged(t *testing.T) {
	for _, kind := range []string{"validation", "write"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if kind == "write" {
				path = filepath.Join(t.TempDir(), "missing", "config.toml")
			}
			bc := conf.Bootstrap{Runtime: conf.Runtime{ConfigPath: path}, Pipeline: conf.Pipeline{WhisperMode: "whisper-cpp"}, TTS: conf.TTS{Type: "edge", Voice: "previous"}, LipSync: conf.LipSync{BaseURL: "previous"}}
			before := bc
			updates := map[string]any{"tts": map[string]any{"type": "openai", "voice": "new"}, "lip_sync": map[string]any{"base_url": "new"}}
			if kind == "validation" {
				updates["pipeline"] = map[string]any{"whisper_mode": "openai"}
			}
			if err := (&AppService{bc: &bc}).UpdateConfig(updates); err == nil {
				t.Fatal("应当保存失败")
			}
			if !reflect.DeepEqual(before, bc) {
				t.Fatal("保存失败不应改变任一内存配置区域")
			}
		})
	}
}
