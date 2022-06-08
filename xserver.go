package cosrpc

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/registry"
	"github.com/smallnest/rpcx/server"
	"reflect"
	"time"
)

// 通过registry集中注册对象

type Register interface {
	Stop() error
	Start() error
	Register(name string, i interface{}, metadata string) (err error)
}

func NewXServer(opts *registry.Options) *XServer {
	r := &XServer{rpcHandler: make(map[string]*RegistryHandler)}
	if opts == nil {
		opts = registry.NewOptions()
	}
	if opts.Filter == nil {
		opts.Filter = r.filter
	}
	r.Registry = registry.New(opts)
	return r
}

type XServer struct {
	*registry.Registry
	Caller      RegistryCaller    //全局消息调用
	Serialize   RegistrySerialize //全局消息序列化封装
	rpcServer   *server.Server
	rpcHandler  map[string]*RegistryHandler
	rpcRegister Register
}

func (this *XServer) filter(pr, fn reflect.Value) bool {
	if !pr.IsValid() {
		_, ok := fn.Interface().(func(*server.Context) interface{})
		return ok
	}
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

//handle cosweb入口
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
	pr, fn, ok := service.Match(urlPath)
	if !ok {
		return errors.New("ServiceMethod not exist")
	}
	handler := this.rpcHandler[service.Name()]

	var reply interface{}
	if handler != nil && handler.Caller != nil {
		reply, err = handler.Caller(sc, pr, fn)
	} else if this.Caller != nil {
		reply, err = this.Caller(sc, pr, fn)
	} else {
		reply, err = this.caller(sc, pr, fn)
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

func (this *XServer) caller(c *server.Context, pr, fn reflect.Value) (reply interface{}, err error) {
	if !pr.IsValid() {
		f, _ := fn.Interface().(func(c *server.Context) interface{})
		reply = f(c)
	} else if s, ok := pr.Interface().(RegistryInterface); ok {
		reply = s.Caller(c, fn)
	} else {
		ret := fn.Call([]reflect.Value{pr, reflect.ValueOf(c)})
		reply = ret[0].Interface()
	}
	return
}

func (this *XServer) RpcServer() *server.Server {
	return this.rpcServer
}

//func (this *XServer) Route(name string) *registry.Service {
//	route := this.Registry.Service(name)
//	return route
//}

//func (this *XServer) Services() (s []string) {
//	this.Registry.Range(func(name string, _ *registry.Service) bool {
//		servicePath := strings.TrimPrefix(name, "/")
//		s = append(s, servicePath)
//		return true
//	})
//	return
//}

func (this *XServer) Service(name string, middleware ...interface{}) *registry.Service {
	s := this.Registry.Service(name)
	if len(middleware) > 0 {
		handler := &RegistryHandler{}
		for _, m := range middleware {
			handler.Use(m)
		}
		this.rpcHandler[s.Name()] = handler
	}
	return s
}

func (this *XServer) Start(address *utils.Address, register Register) (err error) {
	if err = register.Start(); err != nil {
		return
	}
	this.rpcServer = server.NewServer()
	this.rpcServer.DisableHTTPGateway = true
	for _, service := range this.Registry.Services() {
		servicePath := service.Name()
		var metadata string
		if handle, ok := this.rpcHandler[servicePath]; ok {
			if handle.Metadata != nil {
				metadata = handle.Metadata()
			}
		}
		if err = register.Register(servicePath, nil, metadata); err != nil {
			return
		}
		for _, serviceMethod := range service.Paths() {
			this.rpcServer.AddHandler(servicePath, serviceMethod, this.handle)
		}
	}
	if err != nil {
		return
	}
	uri, _ := address.URL("tcp")
	err = utils.Timeout(time.Second, func() error {
		return this.rpcServer.Serve(uri.Scheme, uri.Host)
	})
	if err == utils.ErrorTimeout {
		err = nil
	}
	return
}

func (this *XServer) Close() error {
	this.rpcServer.Shutdown(nil)
	this.rpcRegister.Stop()
	return nil
}
