package web

import (
	"bytes"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ixugo/goddd/pkg/hook"
)

// etagWriter 缓存响应体用于计算 ETag。
type etagWriter struct {
	http.ResponseWriter
	body bytes.Buffer
}

func (w *etagWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w *etagWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// CacheControlMaxAge 设置 Cache-Control max-age 头。
func CacheControlMaxAge(second int, ignoreFn ...IgnoreOption) Middleware {
	age := strconv.Itoa(second)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, fn := range ignoreFn {
				if fn(r) {
					next.ServeHTTP(w, r)
					return
				}
			}
			if r.Method == http.MethodGet {
				w.Header().Set("Cache-Control", "max-age="+age)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// EtagHandler 添加 ETag 头，不适合大文件场景。
func EtagHandler(ignoreFn ...IgnoreOption) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, fn := range ignoreFn {
				if fn(r) {
					next.ServeHTTP(w, r)
					return
				}
			}
			ew := &etagWriter{ResponseWriter: w}
			next.ServeHTTP(ew, r)

			buf := ew.body.Bytes()
			hash := hook.MD5FromBytes(buf)
			etag := `"` + hash + `"`
			w.Header().Set("ETag", etag)
			if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			if _, err := w.Write(buf); err != nil {
				slog.ErrorContext(r.Context(), "write err", "err", err)
			}
		})
	}
}
