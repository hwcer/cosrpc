package cosrpc

import (
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosrpc/redis"
	"github.com/hwcer/cosrpc/xserver"
	"github.com/hwcer/logger"
)

var Server = &rpcServer{XServer: xserver.NewXServer()}

type rpcServer struct {
	*xserver.XServer
	address  *utils.Address
	register *redis.RedisRegisterPlugin
}

type rpcxServiceHandlerMetadata interface {
	Metadata() string
}

func (this *rpcServer) Address() *utils.Address {
	if this.address != nil {
		return this.address
	}
	this.address = utils.NewAddress(Options.Rpcx.Address)
	if cosgo.Debug() && this.address.Retry == 0 {
		this.address.Retry = 100
	}
	if this.address.Host == "" {
		this.address.Host, _ = LocalIpv4()
	}
	this.address.Scheme = Options.Rpcx.Network
	return this.address
}

func (this *rpcServer) Start() (err error) {
	if this.XServer.Registry.Len() == 0 {
		return nil
	}
	err = this.Address().Handle(func(network, address string) error {
		return this.XServer.Start(network, address)
	})
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			logger.Trace("rpc server started %v", this.Address().String())
		}
	}()
	if Options.Rpcx.Redis == "" {
		return
	}
	//注册服务,实现 rpcxServiceHandlerMetadata 才具有服务发现功能
	service := map[string]string{}
	this.XServer.Registry.Range(func(s *registry.Service) bool {
		name := s.Name()
		if mf, ok := s.Handler.(rpcxServiceHandlerMetadata); ok {
			service[name] = mf.Metadata()
		}
		return true
	})
	if len(service) == 0 {
		return
	}
	if this.register, err = getRpcRegister(this.Address(), Options.Rpcx.BasePath); err != nil {
		return
	}
	for name, metadata := range service {
		if err = this.register.Register(name, nil, metadata); err != nil {
			return err
		}
	}
	return this.register.Start()
}

func (this *rpcServer) Close() (err error) {
	if err = this.XServer.Close(); err != nil {
		return
	}
	if this.register != nil {
		err = this.register.Stop()
	}
	return
}
