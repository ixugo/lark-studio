package web

import (
	"expvar"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/ixugo/goddd/pkg/queue"
)

// Metrics 统计请求数、响应数、URL 热度、状态码分布。
func Metrics() Middleware {
	request := expvar.NewInt("request")
	totalRequests := expvar.NewInt("requests")
	totalResponses := expvar.NewInt("responses")
	urls := expvar.NewMap("requestURLs")
	statusCodes := expvar.NewMap("statusCodes")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			totalRequests.Add(1)
			request.Add(1)

			rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(rec, r)

			request.Add(-1)
			totalResponses.Add(1)

			if rec.code != 404 {
				urls.Add(r.Method+" "+r.URL.Path, 1)
			}
			statusCodes.Add(strconv.Itoa(rec.code), 1)
		})
	}
}

// statusRecorder 捕获状态码，供中间件使用。
type statusRecorder struct {
	http.ResponseWriter
	code    int
	written bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.written {
		r.code = code
		r.written = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

type GoroutineNum struct {
	Time string `json:"time"`
	Num  int    `json:"num"`
}

// CountGoroutines 协程数量，间隔 duration 记录一次
func CountGoroutines(d time.Duration, num uint8) {
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	goroutine := queue.NewCirQueue[GoroutineNum](num)

	expvar.Publish("goroutine_num", expvar.Func(func() any {
		return goroutine.Range()
	}))

	for {
		goroutine.Push(GoroutineNum{
			Time: time.Now().Format(time.DateTime),
			Num:  runtime.NumGoroutine(),
		})
		<-ticker.C
	}
}
