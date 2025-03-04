package xserver

import "github.com/hwcer/cosgo/registry"

var Default = New()

func Service(name string, handler ...interface{}) *registry.Service {
	return Default.Service(name, handler...)
}

func Registry() *registry.Registry {
	return Default.Registry
}

func Reload(nodes map[string]*registry.Node) error {
	return Default.Reload(nodes)
}

func Start(register Register) (err error) {
	return Default.Start(register)
}

func Close() (err error) {
	return Default.Close()
}
