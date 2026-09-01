package web

import (
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"
)

// Middleware 标准 net/http 中间件。
type Middleware func(http.Handler) http.Handler

// Chain 按顺序包装中间件，第一个最先执行。
func Chain(handler http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}
	return handler
}

// IgnoreOption 返回 true 表示跳过当前中间件逻辑。
type IgnoreOption func(r *http.Request) bool

// ScrollPageOutput 滚动翻页
type ScrollPageOutput[T any] struct {
	Items []T    `json:"items"`
	Next  string `json:"next"`
}

// PageOutput 分页数据
type PageOutput[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

// PagerFilter 分页过滤
type PagerFilter struct {
	Page         int      `form:"page"`
	Size         int      `form:"size"`
	Sort         string   `form:"sort"`
	SortSafelist []string `json:"-"`
}

func NewPagerFilterMaxSize() PagerFilter {
	return PagerFilter{
		Size: 99999,
	}
}

// DateFilter 日期区间过滤
type DateFilter struct {
	StartMs int64 `form:"start_ms"`
	EndMs   int64 `form:"end_ms"`
}

// StartAt 开始时间
func (d DateFilter) StartAt() time.Time {
	return time.UnixMilli(d.StartMs)
}

// EndAt 结束时间
func (d DateFilter) EndAt() time.Time {
	return time.UnixMilli(d.EndMs)
}

// DefaultStartAt 当为零值或不符合规则时，返回提供的默认值
func (d DateFilter) DefaultStartAt(date time.Time) time.Time {
	if d.StartMs <= 0 || d.StartMs > d.EndMs {
		return date
	}
	return time.UnixMilli(d.StartMs)
}

// DefaultEndAt 当为零值或不符合规则时，返回提供的默认值
func (d DateFilter) DefaultEndAt(date time.Time) time.Time {
	if d.EndMs <= 0 || d.EndMs < d.StartMs {
		return date
	}
	return time.UnixMilli(d.EndMs)
}

// MustSortColumn 忽略安全问题，失败返回空串
func (f PagerFilter) MustSortColumn() string {
	column, ok := f.SortColumn()
	if !ok {
		return ""
	}
	return column + " " + f.SortDirection()
}

// SortColumn 通过对 SortColumn 设置值，仅对允许的值做排序处理
func (f PagerFilter) SortColumn() (string, bool) {
	if f.Sort != "" && slices.Contains(f.SortSafelist, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-"), true
	}
	return "", false
}

// SortDirection 如果 sort 携带负号返回倒序，否则返回正序
func (f PagerFilter) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

// Offset 计算偏离数值
func (f PagerFilter) Offset() int {
	if f.Page < 1 {
		f.Page = 1
	}
	return (f.Page - 1) * f.Size
}

// Limit 每页 1~10000 区间
func (f PagerFilter) Limit() int {
	if f.Size <= 1 {
		return 1
	}
	if f.Size > 10000 {
		return 10000
	}
	return f.Size
}

// Limit 限制数值在 min 和 max 之间
func Limit(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func Offset(page, size int) int {
	if page < 1 {
		return 1
	}
	return (page - 1) * size
}

// GetBaseURL 提取请求地址
func GetBaseURL(req *http.Request) string {
	if v := req.Header.Get("X-Forwarded-Prefix"); v != "" {
		return v
	}
	return fmt.Sprintf("%s://%s", GetScheme(req), req.Host)
}

// BaseURLJoin 拼接 base URL
func BaseURLJoin(req *http.Request, paths ...string) string {
	baseURL := GetBaseURL(req)
	return baseURL + "/" + strings.TrimPrefix(path.Join(paths...), "/")
}

// GetHost 提取主机 IP 或域名
func GetHost(req *http.Request) string {
	if v := req.Header.Get("X-Forwarded-Host"); v != "" {
		return v
	}
	host := req.Host
	if l := strings.Split(host, ":"); len(l) == 2 {
		host = l[0]
	}
	return host
}

// GetScheme 获取请求协议
func GetScheme(req *http.Request) string {
	if v := req.Header.Get("X-Forwarded-Scheme"); v != "" {
		return v
	}
	if req.URL.Scheme != "" {
		return req.URL.Scheme
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

// XForwardedPrefix 解决反向代理路由问题
func XForwardedPrefix(req *http.Request, p string) string {
	return strings.TrimSuffix(req.Header.Get("X-Forwarded-Prefix"), "/") + p
}

// SetDeadline 设置读写超时。
func SetDeadline(dura time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rc := http.NewResponseController(w)
			deadline := time.Now().Add(dura)
			_ = rc.SetWriteDeadline(deadline)
			_ = rc.SetReadDeadline(deadline)
			next.ServeHTTP(w, r)
		})
	}
}
