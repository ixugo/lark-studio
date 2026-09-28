package api

import (
	"expvar"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ixugo/vdub/pkg/web"
	"github.com/ixugo/vdub/pkg/ws"
)

var startRuntime = time.Now()

func setupRouter(mux *http.ServeMux, uc *Usecase) {
	go web.CountGoroutines(10*time.Minute, 20)

	mux.HandleFunc("/health", web.WrapH(uc.getHealth))
	mux.HandleFunc("GET /app/metrics/api", web.WrapH(uc.getMetricsAPI))

	RegisterVersion(mux, uc.Version)
	RegisterTask(mux, uc.TaskAPI)
	RegisterTerm(mux, uc.TermAPI)
	RegisterModel(mux, uc.Hub, uc.Conf)

	mux.HandleFunc("GET /config", web.WrapH(uc.getConfig))
	mux.HandleFunc("PUT /config", web.WrapH(uc.updateConfig))
	mux.HandleFunc("GET /ws", uc.Hub.ServeHTTP)

	startUIWatchdog(uc.Hub)
}

// startUIWatchdog 监控 UI WebSocket 连接。
// 首次连接建立后，若所有连接断开超过 75 秒无重连，则自动退出进程。
func startUIWatchdog(hub ws.Huber) {
	const watchdogTimeout = 75 * time.Second

	var (
		connCount    atomic.Int32
		hadConn      atomic.Bool
		lastDropTime atomic.Value
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

func (uc *Usecase) getHealth(_ *http.Request, _ *struct{}) (getHealthOutput, error) {
	return getHealthOutput{
		Version:   uc.Conf.Runtime.BuildVersion,
		GitBranch: strings.Trim(expvar.Get("git_branch").String(), `"`),
		GitHash:   strings.Trim(expvar.Get("git_hash").String(), `"`),
		StartAt:   startRuntime,
	}, nil
}

type getMetricsAPIOutput struct {
	RealTimeRequests int64  `json:"real_time_requests"`
	TotalRequests    int64  `json:"total_requests"`
	TotalResponses   int64  `json:"total_responses"`
	RequestTop       []KV   `json:"request_top"`
	StatusCodeTop    []KV   `json:"status_code_top"`
	Goroutines       any    `json:"goroutines"`
	NumGC            uint32 `json:"num_gc"`
	SysAlloc         uint64 `json:"sys_alloc"`
	StartAt          string `json:"start_at"`
}

func (uc *Usecase) getMetricsAPI(_ *http.Request, _ *struct{}) (*getMetricsAPIOutput, error) {
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
