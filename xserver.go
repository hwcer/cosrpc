package cosrpc

import (
	"errors"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
	"runtime/debug"
	"time"
)

// Caller struct自带的Caller
type Caller interface {
	Caller(c *server.Context, node *registry.Node) interface{}
}

func NewXServer() *XServer {
	r := &XServer{}
	r.Server = server.NewServer()
	r.Binder = binder.New(binder.MIMEJSON)
	r.Registry = registry.New(nil)
	r.Server.DisableHTTPGateway = true
	return r
}

type XServer struct {
	*server.Server
	jsonrpc  jsonrpcHandle
	Binder   binder.Interface
	Registry *registry.Registry
}

// rpcxHandle 闭包绑定 route和Node
func (xs *XServer) handle(node *registry.Node) func(*server.Context) error {
	return func(sc *server.Context) error {
		return xs.caller(sc, node)
	}
}

// caller services入口
func (xs *XServer) caller(sc *server.Context, node *registry.Node) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	handler, ok := node.Service.Handler.(*Handler)
	if !ok {
		return errors.New("handler unknown")
	}
	c := NewContext(sc, xs.Binder)
	if reply, err := handler.handle(node, c); err != nil {
		return err
	} else if data, err := handler.marshal(c, reply); err != nil {
		return err
	} else {
		return c.ctx.Write(data)
	}
}

//func (this *XServer) Server() *server.Server {
//	return this.rpcServer
//}

func (xs *XServer) Service(name string, handler ...interface{}) *registry.Service {
	service := xs.Registry.Service(name)
	if service.Handler == nil {
		service.Handler = &Handler{}
	}
	if h, ok := service.Handler.(*Handler); ok {
		for _, i := range handler {
			h.Use(i)
		}
	}
	return service
}

func (xs *XServer) Start(network, address string) (err error) {
	//启动服务
	xs.Registry.Nodes(func(node *registry.Node) (r bool) {
		xs.Server.AddHandler(node.Service.Name(), node.Name(), xs.handle(node))
		return true
	})

	//this.Server.Plugins.Add(register)
	err = scc.Timeout(time.Second, func() error {
		return xs.Server.Serve(network, address)
	})
	if errors.Is(scc.ErrorTimeout, err) {
		err = nil
	}
	return
}

func (xs *XServer) Close() error {
	_ = xs.Server.Shutdown(nil)
	return nil
}
