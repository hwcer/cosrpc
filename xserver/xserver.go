package xserver

import (
	"errors"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// Caller struct自带的Caller
type Caller interface {
	Caller(c *server.Context, node *registry.Node) interface{}
}

type Register interface {
	Start() error
	Stop() error
	Register(name string, rcvr interface{}, metadata string) (err error)
}

func New() *XServer {
	r := &XServer{}
	r.Server = server.NewServer()
	r.Registry = registry.New(nil)
	r.Server.DisableHTTPGateway = true
	return r
}

type XServer struct {
	*server.Server
	started  int32
	Register Register
	Registry *registry.Registry
}

// rpcxHandle 闭包绑定 route和Node
//
//	func (xs *XServer) handle(node *registry.Node) func(*server.Context) error {
//		return func(sc *server.Context) error {
//			return xs.Caller(sc, node)
//		}
//	}
func (xs *XServer) handle(sc *server.Context) error {
	node, ok := xs.Registry.Match(sc.ServicePath(), sc.ServiceMethod())
	if !ok {
		return errors.New("service not found")
	}
	return xs.Caller(sc, node)
}

// Caller services入口
func (xs *XServer) Caller(sc xshare.XContext, node *registry.Node) (err error) {
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

// Reload 动态加载，热更
func (xs *XServer) Reload(nodes map[string]*registry.Node) error {
	if err := xs.Registry.Reload(nodes); err != nil {
		return err
	}
	handles := make(map[string]server.Handler)
	for k, _ := range nodes {
		handles[k] = xs.handle
	}
	xs.Server.UpdateHandler(handles)
	return nil
}
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

func (xs *XServer) Start(register Register) (err error) {
	if xs.Registry.Len() == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&xs.started, 0, 1) {
		return
	}
	if register != nil {
		xs.Register = register
	}
	address := xshare.Address()

	//启动服务
	xs.Registry.Nodes(func(node *registry.Node) (r bool) {
		xs.Server.AddHandler(node.Service.Name(), node.Name(), xs.handle)
		return true
	})

	err = address.Handle(func(network, address string) error {
		return xs.startServe(network, address)
	})
	if err != nil {
		return
	}

	if err = xs.startRegister(); err != nil {
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

func (xs *XServer) startRegister() (err error) {
	if xs.Register == nil {
		logger.Alert("register is nil,Can only run in standalone mode")
		return nil
	}
	//注册服务,实现 rpcxServiceHandlerMetadata 才具有服务发现功能
	service := map[string]string{}
	xs.Registry.Range(func(s *registry.Service) bool {
		name := s.Name()
		service[name] = xshare.Metadata.Get(name)
		return true
	})
	if len(service) == 0 {
		return
	}
	for name, metadata := range service {
		if err = xs.Register.Register(name, nil, metadata); err != nil {
			return err
		}
	}
	if err = xs.Register.Start(); err != nil {
		return
	}
	xs.Server.Plugins.Add(xs.Registry)
	return
}

func (xs *XServer) Close() (err error) {
	if !atomic.CompareAndSwapInt32(&xs.started, 1, 0) {
		return
	}
	if err = xs.Server.Shutdown(nil); err != nil {
		return
	}
	if xs.Register != nil {
		err = xs.Register.Stop()
	}
	return
}
