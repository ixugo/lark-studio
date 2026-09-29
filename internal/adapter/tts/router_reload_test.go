package tts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConfigReloadKeepsInFlightRequestAndDoesNotBlock(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.Model == "old-model" {
			started <- struct{}{}
			<-release
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("audio"))
	}))
	defer endpoint.Close()
	defer unblock()
	router := NewRouter("openai", "alloy", endpoint.URL+"/v1", "fixture-old", "old-model")
	result := make(chan error, 1)
	var work sync.WaitGroup
	work.Go(func() { result <- router.Synthesize(t.Context(), "old", filepath.Join(t.TempDir(), "old.mp3"), "") })
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("旧请求未开始")
	}
	updated := make(chan struct{})
	work.Go(func() {
		router.SetTTSConfig("openai", "new-voice", endpoint.URL+"/v1", "fixture-new", "new-model")
		close(updated)
	})
	select {
	case <-updated:
	case <-time.After(2 * time.Second):
		t.Fatal("保存配置等待了进行中的合成请求")
	}
	if err := router.Synthesize(t.Context(), "new", filepath.Join(t.TempDir(), "new.mp3"), ""); err != nil {
		t.Fatal(err)
	}
	unblock()
	work.Wait()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
