package web

import (
	"context"
	"net/http"
)

// setCtxValue 向 context 写入键值对。
func setCtxValue(ctx context.Context, key ctxKey, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

// AddVaryAuth 给响应添加 Vary: Authorization 头。
func AddVaryAuth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Vary", "Authorization")
			next.ServeHTTP(w, r)
		})
	}
}
