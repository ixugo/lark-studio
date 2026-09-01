package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover 从 panic 恢复并返回 500。
func Recover() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					trace := debug.Stack()
					slog.Error("panic", "err", rec, "stack", string(trace))
					traceID, _ := TraceID(r.Context())
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"msg":      fmt.Sprint(rec),
						"trace_id": traceID,
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
