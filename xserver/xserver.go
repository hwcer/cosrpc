package xserver

import (
	"errors"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosrpc/redis"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"github.com/hwcer/scc"
	"github.com/smallnest/rpcx/server"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// Caller struct自带的Caller
type Caller interface {
	Caller(c *server.Context, node *registry.Node) interface{}
}

type rpcxServiceHandlerMetadata interface {
	Metadata() string
}

func New() *XServer {
	r := &XServer{}
	r.Server = server.NewServer()
	//r.Binder = binder.New(binder.MIMEJSON)
	r.Registry = registry.New(nil)
	r.Server.DisableHTTPGateway = true
	return r
}

type XServer struct {
	*server.Server
	start int32
	//Binder   binder.Interface
	Registry *registry.Registry
	register *redis.Register
}

// rpcxHandle 闭包绑定 route和Node
func (xs *XServer) handle(node *registry.Node) func(*server.Context) error {
	return func(sc *server.Context) error {
		return xs.caller(sc, node)
	}
}

// caller services入口
func (xs *XServer) caller(sc *server.Context, node *registry.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	handler, ok := node.Service.Handler.(*xshare.Handler)
	if !ok {
		return errors.New("handler unknown")
	}
	c := xshare.NewContext(sc)
	var reply any
	reply, err = handler.Caller(node, c)
	if err != nil {
		return
	}
	var data []byte
	if data, err = handler.Marshal(c, reply); err == nil {
		return c.Write(data)
	}
	return
}

//func (this *XServer) Server() *server.Server {
//	return this.rpcServer
//}

func (xs *XServer) Service(name string, handler ...interface{}) *registry.Service {
	service := xs.Registry.Service(name)
	if service.Handler == nil {
		service.Handler = &xshare.Handler{}
	}
	if h, ok := service.Handler.(*xshare.Handler); ok {
		for _, i := range handler {
			h.Use(i)
		}
	}
	return service
}

func (xs *XServer) Start() (err error) {
	if xs.Registry.Len() == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&xs.start, 0, 1) {
		return
	}

	address := xshare.Address()

	//启动服务
	xs.Registry.Nodes(func(node *registry.Node) (r bool) {
		xs.Server.AddHandler(node.Service.Name(), node.Name(), xs.handle(node))
		return true
	})

	err = address.Handle(func(network, address string) error {
		return xs.startServe(network, address)
	})
	if err != nil {
		return
	}

	if err = xs.startRegister(address); err != nil {
		return
	}
	logger.Trace("rpc server started:%v", address.String())
	return
}

func (xs *XServer) startServe(network, address string) (err error) {
	//this.Server.Plugins.Add(register)
	err = scc.Timeout(time.Second, func() error {
		return xs.Server.Serve(network, address)
	})
	if errors.Is(scc.ErrorTimeout, err) {
		err = nil
	}
	return
}

func (xs *XServer) startRegister(address *utils.Address) (err error) {
	if xshare.Options.Redis == "" {
		return
	}
	//注册服务,实现 rpcxServiceHandlerMetadata 才具有服务发现功能
	service := map[string]string{}
	xs.Registry.Range(func(s *registry.Service) bool {
		name := s.Name()
		if mf, ok := s.Handler.(rpcxServiceHandlerMetadata); ok {
			service[name] = mf.Metadata()
		}
		return true
	})
	if len(service) == 0 {
		return
	}
	if xs.register, err = xshare.Register(address); err != nil {
		return
	}
	for name, metadata := range service {
		if err = xs.register.Register(name, nil, metadata); err != nil {
			return err
		}
	}
	return xs.register.Start()
}

func (xs *XServer) Close() (err error) {
	if err = xs.Server.Shutdown(nil); err != nil {
		return
	}
	if xs.register != nil {
		err = xs.register.Stop()
	}
	return
}

func (xs *XServer) Address() *utils.Address {
	address := utils.NewAddress(xshare.Options.Address)
	if address.Retry == 0 {
		address.Retry = 100
	}
	if address.Host == "" {
		address.Host, _ = xshare.LocalIpv4()
	}
	address.Scheme = xshare.Options.Network
	return address
}
