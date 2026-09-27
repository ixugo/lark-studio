package whisper

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnsureSpeechModelInstallsEmbeddedData 确保启动资源与打包文件逐字节一致。
func TestEnsureSpeechModelInstallsEmbeddedData(t *testing.T) {
	path, err := EnsureSpeechModel(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, embeddedSpeechModel) {
		t.Fatal("释放到统一资源目录的 Silero 文件与嵌入数据不一致")
	}
}

// TestEnsureSpeechModelRepairsCorruptFile 确保损坏文件由打包资源恢复为有效模型。
func TestEnsureSpeechModelRepairsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, speechModelName), []byte("broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := EnsureSpeechModel(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, embeddedSpeechModel) || strings.Contains(string(data), "broken") {
		t.Fatal("损坏模型未从嵌入资源恢复")
	}
}
