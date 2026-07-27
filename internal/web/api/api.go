package api

import (
	"expvar"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ixugo/goddd/domain/version/versionapi"
	"github.com/ixugo/goddd/pkg/web"
	"github.com/ixugo/vdub/pkg/ws"
)

var startRuntime = time.Now()

func setupRouter(r *gin.Engine, uc *Usecase) {
	r.Use(
		// 格式化输出到控制台，然后记录到日志
		// 此处不做 recover，底层 http.server 也会 recover，但不会输出方便查看的格式
		gin.CustomRecovery(func(c *gin.Context, err any) {
			slog.Error("panic", "err", err, "stack", string(debug.Stack()))
			c.AbortWithStatus(http.StatusInternalServerError)
		}),
		web.Metrics(),
		web.Logger(),
		// debug 环境中配合 debug 日志级别，记录请求体与响应体
		web.LoggerWithBody(web.DefaultBodyLimit, func(_ *gin.Context) bool {
			// true: 表示忽略记录日志
			// !debug 表示非调试环境不记录
			return !uc.Conf.Runtime.Debug
		}),
	)
	go web.CountGoroutines(10*time.Minute, 20)

	auth := web.AuthMiddleware(uc.Conf.Server.HTTP.JwtSecret)
	r.Any("/health", web.WrapH(uc.getHealth))
	r.GET("/app/metrics/api", web.WrapH(uc.getMetricsAPI))

	versionapi.Register(r, uc.Version, auth)
	RegisterTask(r, uc.TaskAPI)

	r.GET("/config", web.WrapH(uc.getConfig))
	r.PUT("/config", web.WrapH(uc.updateConfig))
	r.GET("/ws", gin.WrapF(uc.Hub.ServeHTTP))

	startUIWatchdog(uc.Hub)
}

// startUIWatchdog 监控 UI WebSocket 连接。
// 首次连接建立后，若所有连接断开超过 75 秒无重连，则自动退出进程。
// 用于 Flutter 崩溃/关闭时自动回收 Go 引擎。
func startUIWatchdog(hub ws.Huber) {
	const watchdogTimeout = 75 * time.Second

	var (
		connCount    atomic.Int32
		hadConn      atomic.Bool
		lastDropTime atomic.Value // time.Time
	)

	hub.SetConnectHandler(func(_ *ws.Client) error {
		connCount.Add(1)
		hadConn.Store(true)
		return nil
	})
	hub.SetDisconnectHandler(func(_ *ws.Client, _ error) {
		if connCount.Add(-1) <= 0 {
			lastDropTime.Store(time.Now())
		}
	})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if !hadConn.Load() {
				continue
			}
			if connCount.Load() > 0 {
				continue
			}
			t, ok := lastDropTime.Load().(time.Time)
			if !ok {
				continue
			}
			if time.Since(t) > watchdogTimeout {
				slog.Info("UI 全部断开超过 75s，自动退出", "last_drop", t.Format(time.DateTime))
				os.Exit(0)
			}
		}
	}()
}

type getHealthOutput struct {
	Version   string    `json:"version"`
	StartAt   time.Time `json:"start_at"`
	GitBranch string    `json:"git_branch"`
	GitHash   string    `json:"git_hash"`
}

func (uc *Usecase) getHealth(_ *gin.Context, _ *struct{}) (getHealthOutput, error) {
	return getHealthOutput{
		Version:   uc.Conf.Runtime.BuildVersion,
		GitBranch: strings.Trim(expvar.Get("git_branch").String(), `"`),
		GitHash:   strings.Trim(expvar.Get("git_hash").String(), `"`),
		StartAt:   startRuntime,
	}, nil
}

type getMetricsAPIOutput struct {
	RealTimeRequests int64  `json:"real_time_requests"` // 实时请求数
	TotalRequests    int64  `json:"total_requests"`     // 总请求数
	TotalResponses   int64  `json:"total_responses"`    // 总响应数
	RequestTop       []KV   `json:"request_top"`        // 请求TOP
	StatusCodeTop    []KV   `json:"status_code_top"`    // 状态码TOP
	Goroutines       any    `json:"goroutines"`         // 协程数量
	NumGC            uint32 `json:"num_gc"`             // gc 次数
	SysAlloc         uint64 `json:"sys_alloc"`          // 内存占用
	StartAt          string `json:"start_at"`           // 运行时间
}

func (uc *Usecase) getMetricsAPI(_ *gin.Context, _ *struct{}) (*getMetricsAPIOutput, error) {
	req := expvar.Get("request").(*expvar.Int).Value()
	reqs := expvar.Get("requests").(*expvar.Int).Value()
	resps := expvar.Get("responses").(*expvar.Int).Value()
	urls := expvar.Get(`requestURLs`).(*expvar.Map)
	status := expvar.Get(`statusCodes`).(*expvar.Map)
	u := sortExpvarMap(urls, 15)
	s := sortExpvarMap(status, 15)
	g := expvar.Get("goroutine_num").(expvar.Func)

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return &getMetricsAPIOutput{
		RealTimeRequests: req,
		TotalRequests:    reqs,
		TotalResponses:   resps,
		RequestTop:       u,
		StatusCodeTop:    s,
		Goroutines:       g(),
		NumGC:            stats.NumGC,
		SysAlloc:         stats.Sys,
		StartAt:          startRuntime.Format(time.DateTime),
	}, nil
}

type KV struct {
	Key   string
	Value int64
}

func sortExpvarMap(data *expvar.Map, top int) []KV {
	kvs := make([]KV, 0, 8)
	data.Do(func(kv expvar.KeyValue) {
		kvs = append(kvs, KV{
			Key:   kv.Key,
			Value: kv.Value.(*expvar.Int).Value(),
		})
	})

	sort.Slice(kvs, func(i, j int) bool {
		return kvs[i].Value > kvs[j].Value
	})

	idx := top
	if l := len(kvs); l < top {
		idx = len(kvs)
	}
	return kvs[:idx]
}
