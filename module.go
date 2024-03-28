package cosrpc

import (
	"errors"
	"github.com/hwcer/cosgo"
)

var mod *Module

func New() *Module {
	if mod == nil {
		mod = &Module{
			Module: cosgo.Module{Id: "rpcx"},
		}
	}
	return mod
}

type Module struct {
	cosgo.Module
}

func (this *Module) Init() (err error) {
	return this.Reload()
}

func (this *Module) Start() (err error) {
	//RPCX CLIENT
	if err = Client.Start(); err != nil {
		return
	}
	return
}

func (this *Module) Close() (err error) {
	return Client.Close()
}

func (this *Module) Reload() (err error) {
	if err = cosgo.Config.Unmarshal(Options); err != nil {
		return
	}
	if Options.Rpcx.BasePath == "" {
		return errors.New("rpcx basePath empty")
	}
	return
}
