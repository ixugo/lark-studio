package web

import (
	"bytes"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ixugo/goddd/pkg/logger"
)

const DefaultBodyLimit = 100

// bodyRecorder 捕获响应体，供 body 日志使用。
type bodyRecorder struct {
	http.ResponseWriter
	body  bytes.Buffer
	limit int
	code  int
}

func (w *bodyRecorder) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyRecorder) Write(b []byte) (int, error) {
	if w.limit <= 0 {
		w.body.Write(b)
		return w.ResponseWriter.Write(b)
	}
	remain := w.limit - w.body.Len()
	if remain > 0 {
		w.body.Write(b[:min(len(b), remain)])
	}
	return w.ResponseWriter.Write(b)
}

func (w *bodyRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// IgnoreBool 忽略指定值
func IgnoreBool(v bool) IgnoreOption {
	return func(*http.Request) bool { return v }
}

// IgnoreMethod 忽略指定请求方式
func IgnoreMethod(method string) IgnoreOption {
	return func(r *http.Request) bool { return r.Method == method }
}

// IgnorePrefix 忽略指定路由前缀
func IgnorePrefix(prefix ...string) IgnoreOption {
	return func(r *http.Request) bool {
		for _, p := range prefix {
			if strings.HasPrefix(r.URL.Path, p) {
				return true
			}
		}
		return false
	}
}

// IgnorePath 忽略指定路由
func IgnorePath(path ...string) IgnoreOption {
	return func(r *http.Request) bool {
		for _, p := range path {
			if r.URL.Path == p {
				return true
			}
		}
		return false
	}
}

// Logger 记录 HTTP 请求日志。
func Logger(ignoreFn ...IgnoreOption) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			guid := uuid.New()
			traceID := hex.EncodeToString(guid[:])
			ctx := logger.WithAttrs(r.Context(), slog.String("trace_id", traceID))
			ctx = WithTraceID(ctx, traceID)
			r = r.WithContext(ctx)

			for _, fn := range ignoreFn {
				if fn(r) {
					next.ServeHTTP(w, r)
					return
				}
			}

			now := time.Now()
			rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(rec, r)

			code := rec.code
			query, err := url.PathUnescape(r.URL.RawQuery)
			if err != nil {
				query = r.URL.RawQuery
			}

			out := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"query", query,
				"remoteaddr", r.RemoteAddr,
				"statuscode", code,
				"duration_ms", time.Since(now).Milliseconds(),
			}
			if code >= 200 && code < 400 {
				slog.InfoContext(r.Context(), "OK", out...)
				return
			}
			if !(code == 404 || code == 401) {
				slog.WarnContext(r.Context(), "Bad", out...)
				return
			}
			slog.WarnContext(r.Context(), "Bad", out...)
		})
	}
}

// LoggerWithBody 记录请求体与响应体（debug 级别）。
func LoggerWithBody(limit int, ignoreFn ...IgnoreOption) Middleware {
	maxSize := int64(limit * 3)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxSize {
				next.ServeHTTP(w, r)
				return
			}
			for _, fn := range ignoreFn {
				if fn(r) {
					next.ServeHTTP(w, r)
					return
				}
			}

			var reqBody string
			raw, err := io.ReadAll(r.Body)
			if err == nil {
				l := min(len(raw), limit)
				reqBody = string(raw[:l])
			}
			r.Body = io.NopCloser(bytes.NewReader(raw))

			brw := &bodyRecorder{ResponseWriter: w, limit: limit, code: http.StatusOK}
			next.ServeHTTP(brw, r)

			if brw.code != 404 {
				slog.DebugContext(r.Context(), "body", "req", reqBody, "resp", brw.body.String())
			}
		})
	}
}
