package wails

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/ixugo/vdub/pkg/ws"
)

// WailsEventHub 实现 ws.Huber 接口，将后端进度和日志直推至 Wails3 客户端。
type WailsEventHub struct {
	app *application.App
}

// NewWailsEventHub 创建 Wails 事件桥接器。
func NewWailsEventHub(app *application.App) *WailsEventHub {
	return &WailsEventHub{app: app}
}

// Broadcast 广播消息到前端事件监听器。
func (h *WailsEventHub) Broadcast(msg ws.Message) {
	if h.app == nil || msg == nil {
		return
	}
	raw := msg.Data()
	var payload any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &payload)
	}
	if payload == nil {
		payload = string(raw)
	}
	h.app.Event.Emit(msg.Type(), payload)
}

func (h *WailsEventHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

func (h *WailsEventHub) SendToClient(ctx context.Context, clientID string, message ws.Message) error {
	h.Broadcast(message)
	return nil
}

func (h *WailsEventHub) SendToClientAsync(ctx context.Context, clientID string, message ws.Message) error {
	h.Broadcast(message)
	return nil
}

func (h *WailsEventHub) CloseClient(clientID string) error {
	return nil
}

func (h *WailsEventHub) SendToGroup(ctx context.Context, groupID string, message ws.Message) error {
	h.Broadcast(message)
	return nil
}

func (h *WailsEventHub) SendToGroupAsync(ctx context.Context, groupID string, message ws.Message) error {
	h.Broadcast(message)
	return nil
}

func (h *WailsEventHub) GroupSize(groupID string) int {
	return 1
}

func (h *WailsEventHub) GetClients() []*ws.Client {
	return nil
}

func (h *WailsEventHub) Close() {}

func (h *WailsEventHub) SetAuthHandler(handler ws.AuthHandler)             {}
func (h *WailsEventHub) SetConnectHandler(handler ws.ConnectHandler)       {}
func (h *WailsEventHub) SetDisconnectHandler(handler ws.DisconnectHandler) {}
func (h *WailsEventHub) SetErrorHandler(handler ws.ErrorHandler)           {}
func (h *WailsEventHub) Handle(msgType string, handler ws.Handler)         {}
func (h *WailsEventHub) SetDefaultHandler(handler ws.Handler)              {}
