package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type identityFixture struct {
	identity string
}

func (f *identityFixture) Synthesize(context.Context, string, string, string) error { return nil }
func (f *identityFixture) SpeechIdentity(context.Context, string, float64) string   { return f.identity }
func TestChangedSpeechIdentityClearsOldAudioButUnchangedKeepsIt(t *testing.T) {
	dir := t.TempDir()
	fixture := &identityFixture{identity: "Vivian"}
	core := NewCore(Config{}, nil, nil, fixture)
	job := Job{OutputDir: dir}
	if err := core.ensureSpeechIdentity(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	audioDir := filepath.Join(dir, "audio_segs")
	if err := os.Mkdir(audioDir, 0700); err != nil {
		t.Fatal(err)
	}
	audio := filepath.Join(audioDir, "0.wav")
	if err := os.WriteFile(audio, []byte("old voice"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := core.ensureSpeechIdentity(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(audio); err != nil {
		t.Fatal("相同身份不应删除断点音频")
	}
	fixture.identity = "Serena"
	if err := core.ensureSpeechIdentity(t.Context(), job); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(audio); !os.IsNotExist(err) {
		t.Fatal("新旧音色会混用")
	}
}

type languageFixture struct {
	identityFixture
	language string
}

func (f *languageFixture) SnapshotTask(ctx context.Context, engine, language, voice string) (context.Context, error) {
	f.language = language
	return ctx, nil
}
func TestTaskSpeechLanguageFollowsActualSpokenText(t *testing.T) {
	for _, tc := range []struct {
		mode int
		want string
	}{{ModeDub, "en"}, {ModeDirectDub, "zh"}, {ModeDubOnly, "zh"}} {
		fixture := &languageFixture{identityFixture: identityFixture{identity: "Vivian"}}
		core := NewCore(Config{}, nil, nil, fixture)
		_, err := core.prepareSpeechContext(t.Context(), Job{Mode: tc.mode, SourceLang: "zh", TargetLang: "en", OutputDir: t.TempDir()})
		if err != nil || fixture.language != tc.want {
			t.Fatalf("mode %v language=%s: %v", tc.mode, fixture.language, err)
		}
	}
}
