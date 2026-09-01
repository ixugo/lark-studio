package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SSE 发送事件
type SSE struct {
	Headers map[string]string
	stream  chan Event
	timeout time.Duration
	cancel  context.CancelFunc

	m      sync.Mutex
	closed bool
}

type Event struct {
	ID    string
	Event string
	Data  []byte
}

func NewSSE(length int, timeout time.Duration) *SSE {
	if length <= 0 {
		length = 1024
	}
	return &SSE{
		stream:  make(chan Event, length),
		timeout: timeout,
		closed:  false,
	}
}

func (s *SSE) Publish(v Event) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.closed {
		return
	}
	s.stream <- v
}

// Stop 立即停止发送事件
func (s *SSE) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// Close 确保所有事件被发送完毕
func (s *SSE) Close() {
	s.m.Lock()
	defer s.m.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.stream)
}

func (s *SSE) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(s.timeout))
	_ = rc.SetReadDeadline(time.Now().Add(s.timeout))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for k, v := range s.Headers {
		w.Header().Set(k, v)
	}

	ctx, cancel := context.WithCancel(req.Context())
	s.cancel = cancel

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-s.stream:
			if ev.ID == "" && ev.Event == "" && len(ev.Data) == 0 {
				return
			}
			if len(ev.ID) > 0 {
				_, _ = fmt.Fprintf(w, "id: %s\n", ev.ID)
			}
			if len(ev.Event) > 0 {
				_, _ = fmt.Fprintf(w, "event: %s\n", ev.Event)
			}
			if len(ev.Data) > 0 {
				_, _ = fmt.Fprintf(w, "data: %s\n", ev.Data)
			}
			_, _ = fmt.Fprint(w, "\n")
			if err := rc.Flush(); err != nil {
				slog.ErrorContext(req.Context(), "flush", "err", err)
				return
			}
		}
	}
}

type Chunk struct {
	Total   int    `json:"total"`
	Current int    `json:"current"`
	Success int    `json:"success"`
	Failure int    `json:"failure"`
	Err     string `json:"err,omitempty"`
}

// SendChunk 发送分块数据（纯 net/http 版本）。
func SendChunk(ch <-chan Chunk, w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "text/plain")
	rc := http.NewResponseController(w)
	var zero Chunk
	for {
		v := <-ch
		if v == zero {
			return
		}
		b, _ := json.Marshal(v)
		if _, err := w.Write(append(b, '\n')); err != nil {
			return
		}
		_ = rc.Flush()
	}
}
