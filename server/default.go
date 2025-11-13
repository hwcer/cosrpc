package server

import (
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/registry"
)

var Default = New()

func init() {
	cosgo.On(cosgo.EventTypStarted, Default.Start)
	cosgo.On(cosgo.EventTypClosing, Default.Close)
}

var defaultRegister func() (Register, error)

func Service(name string, handler ...interface{}) *registry.Service {
	return Default.Service(name, handler...)
}

// SetRegister 设置服务注册
func SetRegister(r func() (Register, error)) {
	defaultRegister = r
}
