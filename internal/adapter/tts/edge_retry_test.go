package tts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestEdgeCommandRetriesAndPublishesSuccessfulAudio(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "attempts")
	script := "#!/bin/sh\nprintf x >> \"" + counter + "\"\ncount=$(/bin/cat \"" + counter + "\")\nif [ \"$count\" = xx ]; then printf audio > \"$6\"; exit 0; fi\nprintf ConnectionTimeoutError\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "edge-tts"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	output := filepath.Join(dir, "audio.wav")
	if err := NewEdgeTTS("").Synthesize(ctx, "hello", output, ""); err != nil {
		t.Fatal(err)
	}
	attempts, err := os.ReadFile(counter)
	if err != nil || string(attempts) != "xx" {
		t.Fatalf("命令应在首次失败后重试一次: %q %v", attempts, err)
	}
	if data, err := os.ReadFile(output); err != nil || string(data) != "audio" {
		t.Fatalf("正式音频未发布: %q %v", data, err)
	}
}

func TestEdgeRetryBudgetAndPartialFiles(t *testing.T) {
	for _, succeedAt := range []int{1, 4, 5} {
		t.Run(fmt.Sprint(succeedAt), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "edge-tts"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir)
			destination := filepath.Join(dir, "audio.wav")
			attempts := 0
			var delays []time.Duration
			edge := NewEdgeTTS("")
			edge.run = func(ctx context.Context, binary string, args ...string) ([]byte, error) {
				attempts++
				temporary := args[5]
				if _, err := os.Stat(destination); !os.IsNotExist(err) {
					t.Fatal("部分音频进入正式缓存")
				}
				if err := os.WriteFile(temporary, []byte("audio"), 0600); err != nil {
					t.Fatal(err)
				}
				if attempts == succeedAt {
					return nil, nil
				}
				return []byte("ConnectionTimeoutError"), fmt.Errorf("timeout")
			}
			edge.wait = func(ctx context.Context, d time.Duration) error {
				delays = append(delays, d)
				files, err := filepath.Glob(filepath.Join(dir, ".edge-*"))
				if err != nil || len(files) != 0 {
					t.Fatal("失败的临时音频未清理")
				}
				return nil
			}
			err := edge.Synthesize(t.Context(), "hello", destination, "")
			if attempts != min(succeedAt, 4) {
				t.Fatalf("请求次数=%d", attempts)
			}
			want := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}[:attempts-1]
			if !slices.Equal(delays, want) {
				t.Fatalf("退避间隔=%v", delays)
			}
			if succeedAt <= 4 {
				if err != nil {
					t.Fatal(err)
				}
				if data, err := os.ReadFile(destination); err != nil || string(data) != "audio" {
					t.Fatal("成功音频未发布")
				}
			} else {
				if _, ok := errors.AsType[*RetryExhaustedError](err); !ok || !strings.Contains(err.Error(), "ConnectionTimeoutError") {
					t.Fatalf("重试耗尽错误丢失: %v", err)
				}
				if _, err := os.Stat(destination); !os.IsNotExist(err) {
					t.Fatal("失败留下音频缓存")
				}
			}
		})
	}
}

func TestEdgeCancellationStopsRetry(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "edge-tts"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	edge := NewEdgeTTS("")
	attempts := 0
	edge.run = func(ctx context.Context, binary string, args ...string) ([]byte, error) {
		attempts++
		return nil, fmt.Errorf("timeout")
	}
	edge.wait = func(ctx context.Context, d time.Duration) error { cancel(); return waitEdgeRetry(ctx, d) }
	err := edge.Synthesize(ctx, "hello", filepath.Join(dir, "audio.wav"), "")
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("取消后继续重试: %d %v", attempts, err)
	}
}
