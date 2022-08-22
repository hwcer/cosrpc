package xserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/utils"
	_ "github.com/hwcer/cosrpc/logger"
	"github.com/hwcer/registry"
	"github.com/smallnest/rpcx/server"
	"strings"
	"time"
)

// 通过registry集中注册对象

func NewXServer(opts *registry.Options) *XServer {
	r := &XServer{rpcHandler: make(map[string]*Handler)}
	if opts == nil {
		opts = registry.NewOptions()
	}
	if opts.Filter == nil {
		opts.Filter = r.filter
	}
	r.Registry = registry.New(opts)
	r.rpcServer = server.NewServer()
	return r
}

type XServer struct {
	*registry.Registry
	Caller     func(c *server.Context, node *registry.Node) (interface{}, error) //全局消息调用
	Serialize  func(c *server.Context, reply interface{}) error                  //全局消息序列化封装
	rpcServer  *server.Server
	rpcHandler map[string]*Handler
	//rpcRegister server.RegisterPlugin
}

func (this *XServer) filter(s *registry.Service, node *registry.Node) bool {
	handler := this.rpcHandler[s.Name()]
	if handler != nil && handler.Filter != nil {
		return handler.Filter(s, node)
	}
	if !node.IsFunc() {
		_, ok := node.Method().(func(*server.Context) interface{})
		return ok
	}
	fn := node.Value()
	t := fn.Type()
	if t.NumIn() != 2 {
		return false
	}
	if t.NumOut() != 1 {
		return false
	}
	//argType := t.In(1)
	//if !argType.Implements(typeOfContext) {
	//	return false
	//}
	return true
}

// handle services入口
func (this *XServer) handle(sc *server.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
			//logger.Error("%v", err)
		}
	}()
	urlPath := this.Registry.Clean(sc.ServicePath(), sc.ServiceMethod())
	service, ok := this.Match(urlPath)
	if !ok {
		return errors.New("ServicePath not exist")
	}
	node, ok := service.Match(urlPath)
	if !ok {
		return errors.New("ServiceMethod not exist")
	}
	handler := this.rpcHandler[service.Name()]

	var reply interface{}
	if handler != nil && handler.Caller != nil {
		reply, err = handler.Caller(sc, node)
	} else if this.Caller != nil {
		reply, err = this.Caller(sc, node)
	} else {
		reply, err = this.caller(sc, node)
	}
	if err != nil {
		return
	}

	if handler != nil && handler.Serialize != nil {
		return handler.Serialize(sc, reply)
	} else if this.Serialize != nil {
		return this.Serialize(sc, reply)
	} else {
		return sc.Write(reply)
	}
}

func (this *XServer) caller(c *server.Context, node *registry.Node) (reply interface{}, err error) {
	if node.IsFunc() {
		f, _ := node.Method().(func(c *server.Context) interface{})
		reply = f(c)
	} else if s, ok := node.Binder().(registryInterface); ok {
		reply = s.Caller(c, node)
	} else {
		ret := node.Call(c)
		reply = ret[0].Interface()
	}
	return
}

func (this *XServer) Server() *server.Server {
	return this.rpcServer
}

func (this *XServer) Service(name string, handlers ...interface{}) *registry.Service {
	s := this.Registry.Service(name)
	if len(handlers) > 0 {
		handler := &Handler{}
		for _, m := range handlers {
			handler.Use(m)
		}
		this.rpcHandler[s.Name()] = handler
	}
	return s
}

func (this *XServer) Start(network, address string, register server.RegisterPlugin) (err error) {
	this.rpcServer.DisableHTTPGateway = true
	for _, service := range this.Registry.Services() {
		servicePath := service.Name()
		var metadata []string
		if handle := this.rpcHandler[servicePath]; handle != nil {
			for _, f := range handle.Metadata {
				metadata = append(metadata, f())
			}
		}
		if err = register.Register(servicePath, nil, strings.Join(metadata, "&")); err != nil {
			return
		}
		for _, serviceMethod := range service.Paths() {
			this.rpcServer.AddHandler(servicePath, serviceMethod, this.handle)
		}
	}
	this.rpcServer.Plugins.Add(register)
	err = utils.Timeout(time.Second, func() error {
		return this.rpcServer.Serve(network, address)
	})
	if err == utils.ErrorTimeout {
		err = nil
	}
	return
}

func (this *XServer) Close() error {
	_ = this.rpcServer.Shutdown(nil)
	//_ = this.rpcRegister.Stop()
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
