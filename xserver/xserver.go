package xserver

import (
	"errors"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosrpc/redis"
	"github.com/hwcer/cosrpc/xshare"
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
	r.Registry = registry.New(nil)
	r.Server.DisableHTTPGateway = true
	return r
}

type XServer struct {
	*server.Server
	started  int32
	register *redis.Register
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

func (xs *XServer) Start() (err error) {
	if xs.Registry.Len() == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&xs.started, 0, 1) {
		return
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
	if !atomic.CompareAndSwapInt32(&xs.started, 1, 0) {
		return
	}
	if err = xs.Server.Shutdown(nil); err != nil {
		return
	}
	if xs.register != nil {
		err = xs.register.Stop()
	}
	return
}
