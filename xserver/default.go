package xserver

import "github.com/hwcer/registry"

var Default = New()

func Service(name string, handler ...interface{}) *registry.Service {
	return Default.Service(name, handler...)
}

func Start() (err error) {
	return Default.Start()
}

func Close() (err error) {
	return Default.Close()
}
