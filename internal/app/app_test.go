package app

import (
	"net"
	"testing"
)

// TestListenLoopback 验证零端口由系统分配，且监听范围限定在本机 IPv4 回环地址。
func TestListenLoopback(t *testing.T) {
	listener, err := listenLoopback(0)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("监听地址类型 = %T", listener.Addr())
	}
	if addr.Port == 0 {
		t.Fatal("系统未分配端口")
	}
	if !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("监听地址 = %s，期望 127.0.0.1", addr.IP)
	}
}
