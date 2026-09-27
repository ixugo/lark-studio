package whisper

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// ProbeFastestURL 并发探测传入的一组 URL，自动选择连接最快、连通正常的节点。
// 若无节点正常响应或超时，默认回退至传入切片的首个备选地址。
func ProbeFastestURL(ctx context.Context, urls []string) string {
	if len(urls) == 0 {
		return ""
	}
	if len(urls) == 1 {
		return urls[0]
	}

	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	fastestCh := make(chan string, 1)
	var once sync.Once
	var wg sync.WaitGroup

	for _, targetURL := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			req, err := http.NewRequestWithContext(probeCtx, http.MethodHead, u, nil)
			if err != nil {
				return
			}
			client := &http.Client{
				Timeout: 1500 * time.Millisecond,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) >= 3 {
						return http.ErrUseLastResponse
					}
					return nil
				},
			}
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				// 2xx 或 3xx 均视为可用通道
				if resp.StatusCode < 400 || resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusTemporaryRedirect {
					once.Do(func() {
						fastestCh <- u
					})
				}
			}
		}(targetURL)
	}

	go func() {
		wg.Wait()
		// 若所有请求均失败，回退发送首个备选
		once.Do(func() {
			fastestCh <- urls[0]
		})
	}()

	select {
	case fastest := <-fastestCh:
		return fastest
	case <-probeCtx.Done():
		return urls[0]
	}
}
