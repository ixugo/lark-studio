package web

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"runtime"
	"slices"
)

// SetupPProf 在 mux 上注册 pprof 路由，仅允许指定 IP 访问。
func SetupPProf(mux *http.ServeMux, ips *[]string) {
	check := debugAccess(ips)
	mux.Handle("GET /debug/pprof/", check(http.HandlerFunc(pprof.Index)))
	mux.Handle("GET /debug/pprof/cmdline", check(http.HandlerFunc(pprof.Cmdline)))
	mux.Handle("GET /debug/pprof/profile", check(http.HandlerFunc(pprof.Profile)))
	mux.Handle("GET /debug/pprof/symbol", check(http.HandlerFunc(pprof.Symbol)))
	mux.Handle("POST /debug/pprof/symbol", check(http.HandlerFunc(pprof.Symbol)))
	mux.Handle("GET /debug/pprof/trace", check(http.HandlerFunc(pprof.Trace)))
	mux.Handle("GET /debug/pprof/allocs", check(pprof.Handler("allocs")))
	mux.Handle("GET /debug/pprof/block", check(pprof.Handler("block")))
	mux.Handle("GET /debug/pprof/goroutine", check(pprof.Handler("goroutine")))
	mux.Handle("GET /debug/pprof/heap", check(pprof.Handler("heap")))
	mux.Handle("GET /debug/pprof/mutex", check(pprof.Handler("mutex")))
	mux.Handle("GET /debug/pprof/threadcreate", check(pprof.Handler("threadcreate")))
	mux.Handle("GET /debug/vars", expvar.Handler())
}

// debugAccess 授权指定 IP 访问 debug 路由。
func debugAccess(ips *[]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lips := *ips
			if len(lips) == 0 {
				next.ServeHTTP(w, r)
				return
			}
			if slices.Contains(lips, r.RemoteAddr) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}

// SetupMutexProfile 启用互斥锁采样
func SetupMutexProfile(rate int) {
	runtime.SetBlockProfileRate(rate)
	runtime.SetMutexProfileFraction(rate)
}
