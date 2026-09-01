package web

import (
	"context"
)

type ctxKey string

const traceIDKey ctxKey = "TRACE_ID_KEY"

// MustTraceID 从 context 获取 trace_id，不存在时 panic。
func MustTraceID(ctx context.Context) string {
	v := ctx.Value(traceIDKey)
	return v.(string)
}

// TraceID 从 context 获取 trace_id。
func TraceID(ctx context.Context) (string, bool) {
	v := ctx.Value(traceIDKey)
	if v == nil {
		return "", false
	}
	return v.(string), true
}

// WithTraceID 将 trace_id 写入 context。
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}
