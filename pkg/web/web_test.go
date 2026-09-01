package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ixugo/goddd/pkg/logger"
)

func TestLogger(t *testing.T) {
	_, _ = logger.SetupSlog(logger.Config{
		Debug: true,
		Level: "debug",
	})

	handler := WrapH(func(r *http.Request, _ *struct{}) (map[string]any, error) {
		slog.InfoContext(r.Context(), "request", "path", r.URL.Path)
		return map[string]any{"message": "Hello, World!"}, nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /a/{id}", handler)
	wrapped := Chain(mux, Logger(), LoggerWithBody(DefaultBodyLimit))

	req := httptest.NewRequest(http.MethodGet, "/a/123", bytes.NewBufferString("h=hello"))
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
}

func TestBaseURLJoin(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080", nil)
	s := BaseURLJoin(req, "/a/b/", "/c/d")
	if s != "http://127.0.0.1:8080/a/b/c/d" {
		t.Errorf("BaseURLJoin() = %s, want %s", s, "http://127.0.0.1:8080/a/b/c/d")
	}

	s = BaseURLJoin(req, "a/b/", "c/d")
	if s != "http://127.0.0.1:8080/a/b/c/d" {
		t.Errorf("BaseURLJoin() = %s, want %s", s, "http://127.0.0.1:8080/a/b/c/d")
	}

	s = BaseURLJoin(req, "a/b", "c/d")
	if s != "http://127.0.0.1:8080/a/b/c/d" {
		t.Errorf("BaseURLJoin() = %s, want %s", s, "http://127.0.0.1:8080/a/b/c/d")
	}

	s = BaseURLJoin(req, "/a/b", "/c/d")
	if s != "http://127.0.0.1:8080/a/b/c/d" {
		t.Errorf("BaseURLJoin() = %s, want %s", s, "http://127.0.0.1:8080/a/b/c/d")
	}

	s = BaseURLJoin(req, "//a/b", "//c/d")
	if s != "http://127.0.0.1:8080/a/b/c/d" {
		t.Errorf("BaseURLJoin() = %s, want %s", s, "http://127.0.0.1:8080/a/b/c/d")
	}

	s = BaseURLJoin(req, "/a/b", "c//d")
	if s != "http://127.0.0.1:8080/a/b/c/d" {
		t.Errorf("BaseURLJoin() = %s, want %s", s, "http://127.0.0.1:8080/a/b/c/d")
	}
}
