package cosrpc

import (
	"fmt"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/log"
)

func init() {
	l := &dummyLogger{}
	log.SetLogger(l)
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
	logger.Error("%v", fmt.Sprint(v...))
}

func (l *dummyLogger) Errorf(format string, v ...any) {
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
