package cosrpc

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/log"
)

func init() {
	l := &dummyLogger{}
	log.SetLogger(l)
}

// disconnectPhrases rpcx 常把错误格式化进字符串,errors.As 无从下手,按特征短语兜底
var disconnectPhrases = []string{
	"use of closed network connection",
	"connection reset by peer",
	"forcibly closed",         //Windows wsarecv: An existing connection was forcibly closed
	"aborted by the software", //Windows wsaeconnaborted
	"connection is closed",   //rpcx 连接池关闭中的常见文案
	"broken pipe",            //对端已关闭后再写
}

// isDisconnectLog rpcx 经本桥上报的 Error 里,对端断开类(客户端杀进程/断网/关停潮)
// 是长连接日常而非故障——降 Debug 只留开发期排查线索,不再进 Error 刷屏。
// 与 cosnet.isDisconnectError 同规(那边是错误对象,这边多了已格式化字符串的兜底)
func isDisconnectLog(v ...any) bool {
	for _, i := range v {
		switch e := i.(type) {
		case error:
			if isDisconnectErr(e) {
				return true
			}
		case string:
			for _, p := range disconnectPhrases {
				if strings.Contains(e, p) {
					return true
				}
			}
		}
	}
	return false
}

func isDisconnectErr(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) && (errno == 10054 || errno == 10053) {
		return true //WSAECONNRESET / WSAECONNABORTED(Windows 上 syscall.ECONNRESET 匹配不到,见 cosnet 同名函数注释)
	}
	return errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE)
}

type dummyLogger struct {
}

func (l *dummyLogger) Debug(v ...any) {
	logger.Debug(fmt.Sprint(v...))
}
func (l *dummyLogger) Debugf(format string, v ...any) {
	logger.Debug(format, v...)
}

func (l *dummyLogger) Info(v ...any) {
	logger.Trace(fmt.Sprint(v...))
}
func (l *dummyLogger) Infof(format string, v ...any) {
	logger.Trace(format, v...)
}

func (l *dummyLogger) Warn(v ...any) {
	logger.Alert(fmt.Sprint(v...))
}
func (l *dummyLogger) Warnf(format string, v ...any) {
	logger.Alert(format, v...)
}

func (l *dummyLogger) Error(v ...any) {
	if isDisconnectLog(v...) {
		logger.Debug("rpcx disconnect: %v", fmt.Sprint(v...))
		return
	}
	logger.Error("%v", fmt.Sprint(v...))
}
func (l *dummyLogger) Errorf(format string, v ...any) {
	if isDisconnectLog(format) || isDisconnectLog(v...) {
		logger.Debug("rpcx disconnect: "+format, v...)
		return
	}
	logger.Error(format, v...)
}

func (l *dummyLogger) Fatal(v ...any) {
	logger.Error("%v", fmt.Sprint(v...))
}
func (l *dummyLogger) Fatalf(format string, v ...any) {
	logger.Error(format, v...)
}

func (l *dummyLogger) Panic(v ...any) {
	logger.Error("%v", fmt.Sprint(v...))
}
func (l *dummyLogger) Panicf(format string, v ...any) {
	logger.Error(format, v...)
}
