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

func Service(name string, handler ...interface{}) *registry.Service {
	return Default.Service(name, handler...)
}

func Reload(nodes map[string]*registry.Node) error {
	return Default.Reload(nodes)
}

// GetRegistry 获取API注册器
func GetRegistry() *registry.Registry {
	return Default.Registry
}

// SetRegister 设置服务注册
func SetRegister(r Register) {
	Default.Register = r
}
