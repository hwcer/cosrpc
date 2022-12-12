package cosrpc

import (
	"crypto/tls"
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
type Register interface {
	server.RegisterPlugin
	Stop() error
	Start() error
}

func NewXServer() *XServer {
	r := &XServer{}
	r.Binder = binder.New(binder.MIMEJSON)
	r.Registry = registry.New(nil)
	r.rpcServer = server.NewServer()
	return r
}

type XServer struct {
	*registry.Registry
	Binder      binder.Interface
	rpcServer   *server.Server
	rpcRegister Register
}

// closure 闭包绑定 route和Node
func (h *XServer) closure(node *registry.Node) func(*server.Context) error {
	return func(sc *server.Context) error {
		return h.handle(sc, node)
	}
}

// handle services入口
func (this *XServer) handle(sc *server.Context, node *registry.Node) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Info("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	service := node.Service()
	handler, ok := service.Handler.(*Handler)
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

func (this *XServer) Server() *server.Server {
	return this.rpcServer
}

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

func (this *XServer) Start(network, address string, register Register) (err error) {
	this.rpcServer.DisableHTTPGateway = true
	//启动服务
	this.Registry.Range(func(service *registry.Service, node *registry.Node) bool {
		this.rpcServer.AddHandler(service.Name(), node.Name(), this.closure(node))
		return true
	})
	this.rpcServer.Plugins.Add(register)
	err = utils.Timeout(time.Second, func() error {
		return this.rpcServer.Serve(network, address)
	})
	if err == utils.ErrorTimeout {
		err = nil
	}
	if err != nil {
		return
	}
	//注册服务
	for _, service := range this.Registry.Services() {
		servicePath := service.Name()
		var metadata string
		if handle, ok := service.Handler.(*Handler); ok {
			metadata = handle.Metadata()
		}
		if err = register.Register(servicePath, nil, metadata); err != nil {
			return
		}
	}
	if err = register.Start(); err != nil {
		_ = this.rpcServer.Shutdown(nil)
		return err
	}
	this.rpcRegister = register
	return
}

func (this *XServer) Close() error {
	_ = this.rpcServer.Shutdown(nil)
	_ = this.rpcRegister.Stop()
	return nil
}

// WithTLSConfig sets tls.Config.
func (this *XServer) WithTLSConfig(cfg *tls.Config) {
	server.WithTLSConfig(cfg)(this.rpcServer)
}

// WithReadTimeout sets readTimeout.
func (this *XServer) WithReadTimeout(readTimeout time.Duration) {
	server.WithReadTimeout(readTimeout)(this.rpcServer)
}

// WithWriteTimeout sets writeTimeout.
func (this *XServer) WithWriteTimeout(writeTimeout time.Duration) {
	server.WithWriteTimeout(writeTimeout)(this.rpcServer)
}
