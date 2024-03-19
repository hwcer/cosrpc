package cosrpc

import (
	"errors"
	"github.com/hwcer/cosgo"
)

var mod *module

func New() *module {
	if mod == nil {
		mod = &module{
			Module: cosgo.Module{Id: "rpcx"},
		}
	}
	return mod
}

type module struct {
	cosgo.Module
}

func (this *module) Init() (err error) {
	return this.Reload()
}

func (this *module) Start() (err error) {
	//RPCX SERVER
	if err = Server.Start(); err != nil {
		return
	}
	//RPCX CLIENT
	if err = Client.Start(); err != nil {
		return
	}
	return
}

func (this *module) Close() (err error) {
	_ = Client.Close()
	_ = Server.Close()
	return
}

func (this *module) Reload() (err error) {
	if err = cosgo.Config.Unmarshal(Options); err != nil {
		return
	}
	if Options.Rpcx.BasePath == "" {
		return errors.New("rpcx basePath empty")
	}
	return
}
