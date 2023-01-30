package cosrpc

import (
	"errors"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/utils"
	"github.com/smallnest/rpcx/server"
	"runtime/debug"
	"time"
)

// Caller struct自带的Caller
type Caller interface {
	Caller(c *server.Context, node *registry.Node) interface{}
}

// Register 通过registry集中注册对象
//type Register interface {
//	server.RegisterPlugin
//	Stop() error
//	Start() error
//}

func NewXServer() *XServer {
	r := &XServer{}
	r.Server = server.NewServer()
	r.Binder = binder.New(binder.MIMEJSON)
	r.Registry = registry.New(nil)
	return r
}

type XServer struct {
	*server.Server
	Binder   binder.Interface
	Registry *registry.Registry
}

// rpcxHandle 闭包绑定 route和Node
func (h *XServer) rpcxHandle(node *registry.Node) func(*server.Context) error {
	return func(sc *server.Context) error {
		return h.handle(sc, node)
	}
}

// httpHandle 闭包绑定 route和Node
//func (h *XServer) httpHandle(node *registry.Node) func(ctx context.Context, args []byte, reply []byte) error {
//	return func(ctx context.Context, args []byte, reply []byte) error {
//		return h.handle(sc, node)
//	}
//}

// handle services入口
func (this *XServer) handle(sc *server.Context, node *registry.Node) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Info("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	handler, ok := node.Service.Handler.(*Handler)
	if !ok {
		return errors.New("handler unknown")
	}
	c := &Context{Context: sc, Binder: this.Binder}
	reply, err := handler.handle(node, c)
	if err != nil {
		return err
	}
	return handler.Serialize(c, reply)
}

//func (this *XServer) Server() *server.Server {
//	return this.rpcServer
//}

func (this *XServer) Service(name string, handler ...interface{}) *registry.Service {
	service := this.Registry.Service(name)
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

func (this *XServer) Start(network, address string) (err error) {
	this.Server.DisableHTTPGateway = true
	//启动服务
	this.Registry.Nodes(func(node *registry.Node) (r bool) {
		//if err = this.Server.RegisterFunctionName(node.Service.Name(), node.Name(), this.closure(node), ""); err != nil {
		//	return false
		//}
		this.Server.AddHandler(node.Service.Name(), node.Name(), this.rpcxHandle(node))
		return true
	})
	if err != nil {
		return
	}
	//this.Server.Plugins.Add(register)
	err = utils.Timeout(time.Second, func() error {
		return this.Server.Serve(network, address)
	})
	if err == utils.ErrorTimeout {
		err = nil
	}
	if err != nil {
		return
	}
	//注册服务
	//this.Registry.Range(func(service *registry.Service) bool {
	//	servicePath := service.Name()
	//	var metadata string
	//	if handle, ok := service.Handler.(*Handler); ok {
	//		metadata = handle.Metadata()
	//	}
	//	if err = register.Register(servicePath, nil, metadata); err != nil {
	//		return false
	//	}
	//	return true
	//})
	//if err != nil {
	//	return
	//}
	//if err = register.Start(); err != nil {
	//	return err
	//}
	//this.Register = register
	return
}

func (this *XServer) Close() error {
	_ = this.Server.Shutdown(nil)
	return nil
}
