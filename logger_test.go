package cosrpc

import (
	"net"
	"syscall"
	"testing"
)

// TestIsDisconnectLog rpcx 经 dummyLogger 上报的关停/断开类错误必须被识别降级,
// 其余错误不得误伤(仍走 logger.Error)。
func TestIsDisconnectLog(t *testing.T) {
	//错误对象形态:net.OpError 包装 WSAECONNRESET(10054),线上实际样子
	if !isDisconnectLog(&net.OpError{Op: "read", Err: syscall.Errno(10054)}) {
		t.Error("应识别 Errno(10054) 包装错误")
	}
	//rpcx 已格式化成字符串的形态
	for _, s := range []string{
		"read tcp 127.0.0.1:8000->127.0.0.1:51674: wsarecv: An existing connection was forcibly closed by the remote host.",
		"accept tcp 0.0.0.0:9000: use of closed network connection",
		"fail to read request header: read tcp ...: connection reset by peer",
	} {
		if !isDisconnectLog(s) {
			t.Errorf("应识别断开类文案: %s", s)
		}
	}
	//真错误不得被过滤
	for _, v := range []any{"some real failure", "connection refused by config"} {
		if isDisconnectLog(v) {
			t.Errorf("不应误伤: %v", v)
		}
	}
}
