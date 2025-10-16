package server

import (
	"errors"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosrpc"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
)

const RegistryMethod = "RPCX"

// Caller struct自带的Caller
type Caller interface {
	Caller(c *cosrpc.Context, node *registry.Node) interface{}
}

type Register interface {
	Start() error
	Stop() error
	Register(name string, rcvr interface{}, metadata string) (err error)
}

func New() *Server {
	r := &Server{}
	r.Server = server.NewServer()
	r.Registry = registry.New()
	r.Server.DisableHTTPGateway = true
	return r
}

type Server struct {
	*server.Server
	started  int32
	Register Register
	Registry *registry.Registry
}

//func (xs *Server) handle(sc *server.Context, node *registry.Node) error {
//	node, ok := xs.Registry.Match(sc.ServicePath(), sc.ServiceMethod())
//	if !ok {
//		return errors.New("service not found")
//	}
//	return xs.Caller(sc, node)
//}

// Caller services入口
func (xs *Server) Caller(sc cosrpc.ICtx, node *registry.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()

	handler, ok := node.Handler().(*Handler)
	if !ok {
		return errors.New("handler unknown")
	}
	c := cosrpc.NewContext(sc)
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

//	func (xs *Server) Reload(nodes map[string]*registry.Node) error {
//		if err := xs.Registry.Reload(nodes); err != nil {
//			return err
//		}
//		handles := make(map[string]server.Handler)
//		for k, _ := range nodes {
//			handles[k] = xs.handle
//		}
//		xs.Server.UpdateHandler(handles)
//		return nil
//	}
func (xs *Server) Service(name string, handlers ...any) *registry.Service {
	handler := &Handler{}
	service := xs.Registry.Service(name, handler)
	service.SetMethods([]string{RegistryMethod})
	for _, i := range handlers {
		handler.Use(i)
	}
	return service
}

func (xs *Server) startServer(network, address string) (err error) {
	//this.Server.Plugins.Add(register)
	err = scc.Timeout(time.Second, func() error {
		return xs.Server.Serve(network, address)
	})
	if errors.Is(scc.ErrorTimeout, err) {
		err = nil
	}
	return
}

func (xs *Server) startRegister() (err error) {
	if xs.Register == nil {
		logger.Alert("register is nil,Can only run in standalone mode")
		return nil
	}
	//注册服务,实现 rpcxServiceHandlerMetadata 才具有服务发现功能
	service := map[string]string{}
	xs.Registry.Range(func(s *registry.Service) bool {
		name := s.Name()
		service[name] = Metadata.Get(name)
		return true
	})
	if len(service) == 0 {
		return
	}
	for name, meta := range service {
		if err = xs.Register.Register(name, nil, meta); err != nil {
			return err
		}
	}
	if err = xs.Register.Start(); err != nil {
		return
	}
	xs.Server.Plugins.Add(xs.Registry)
	return
}

func (xs *Server) parseServiceName(name string) (servicePath string, serviceMethod string) {
	name = strings.TrimPrefix(name, "/")
	i := strings.Index(name, "/")
	servicePath = name[:i]
	serviceMethod = name[i:]
	return
}
func (xs *Server) Start() (err error) {
	if xs.Registry.Len() == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&xs.started, 0, 1) {
		return
	}
	address := cosrpc.Address()
	//启动服务
	xs.Registry.Nodes(func(node *registry.Node) (r bool) {
		servicePath, serviceMethod := xs.parseServiceName(node.Name())
		var handler = func(c *server.Context) error {
			return xs.Caller(c, node)
		}
		xs.Server.AddHandler(servicePath, serviceMethod, handler)
		return true
	})

	err = address.Handle(func(network, address string) error {
		return xs.startServer(network, address)
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

func (xs *Server) Close() (err error) {
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
