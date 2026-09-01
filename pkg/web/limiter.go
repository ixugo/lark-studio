package web

import (
	"net/http"
	"time"

	"github.com/ixugo/goddd/pkg/conc"
	"github.com/ixugo/goddd/pkg/reason"
	"golang.org/x/time/rate"
)

// RateLimiter 全局限流中间件。
func RateLimiter(r rate.Limit, b int, ignoreFn ...IgnoreOption) Middleware {
	l := rate.NewLimiter(r, b)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !l.Allow() {
				for _, fn := range ignoreFn {
					if fn(req) {
						next.ServeHTTP(w, req)
						return
					}
				}
				WriteError(w, req, reason.ErrRateLimit.SetMsg("服务器繁忙"))
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

// IPRateLimiter 按 IP 限流。
func IPRateLimiter(r rate.Limit, b int, ignoreFn ...IgnoreOption) Middleware {
	limiter := IDRateLimiter(r, b, 3*time.Minute)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if !limiter(req.RemoteAddr) {
				for _, fn := range ignoreFn {
					if fn(req) {
						next.ServeHTTP(w, req)
						return
					}
				}
				WriteError(w, req, reason.ErrRateLimit)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

// IDRateLimiter 按标识限流
func IDRateLimiter(r rate.Limit, b int, ttl time.Duration) func(identifier string) bool {
	if ttl == 0 {
		ttl = 3 * time.Minute
	}
	cache := conc.NewTTLMap[string, *rate.Limiter]()
	return func(identifier string) bool {
		v, ok := cache.Load(identifier)
		if !ok {
			v, _ = cache.LoadOrStore(identifier, rate.NewLimiter(r, b), ttl)
		}
		return v.Allow()
	}
}

// LimitContentLength 限制请求体大小。
func LimitContentLength(limit int, ignoreFn ...IgnoreOption) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > int64(limit) {
				for _, fn := range ignoreFn {
					if fn(r) {
						next.ServeHTTP(w, r)
						return
					}
				}
				WriteError(w, r, reason.ErrContentTooLarge)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
