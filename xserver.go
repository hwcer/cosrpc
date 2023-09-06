package cosrpc

import (
	"errors"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosrpc/jsonrpc"
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

func (xs *XServer) Jsonrpc(servicePath, serviceMethod string, handle jsonrpcHandle) {
	xs.jsonrpc = handle
	xs.Server.AddHandler(servicePath, serviceMethod, xs.jsonHandle)
}

// rpcxHandle 闭包绑定 route和Node
func (xs *XServer) rpcxHandle(node *registry.Node) func(*server.Context) error {
	return func(sc *server.Context) error {
		return xs.handle(sc, node)
	}
}

// jsonHandle jsonrpc handle
func (xs *XServer) jsonHandle(sc *server.Context) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	c := NewContext(sc, xs.Binder)
	b := c.Bytes()
	var err error
	var reply []*jsonrpc.Reply
	if string(b[0:1]) == "[" {
		var args []*jsonrpc.Args
		if err = xs.Binder.Unmarshal(b, &args); err != nil {
			return c.ctx.Write(jsonrpc.NewError(-32700, err).Bytes(xs.Binder))
		} else if len(args) == 0 {
			return c.ctx.Write(jsonrpc.NewError(-32600, "Invalid Request").Bytes(xs.Binder))
		} else if reply, err = xs.jsonrpc(c, args); err != nil {
			return c.ctx.Write(jsonrpc.NewError(-32603, err).Bytes(xs.Binder))
		} else if len(reply) != len(args) {
			return c.ctx.Write(jsonrpc.NewError(-32603, "handle reply error").Bytes(xs.Binder))
		} else {
			var data []byte
			if data, err = xs.Binder.Marshal(reply); err != nil {
				return c.ctx.Write(jsonrpc.NewError(-32603, "marshal reply error").Bytes(xs.Binder))
			} else {
				return c.ctx.Write(data)
			}
		}
	} else if string(b[0:1]) == "{" {
		args := &jsonrpc.Args{}
		if err = xs.Binder.Unmarshal(b, args); err != nil {
			return c.ctx.Write(jsonrpc.NewError(-32700, err).Bytes(xs.Binder))
		} else if reply, err = xs.jsonrpc(c, []*jsonrpc.Args{args}); err != nil {
			return c.ctx.Write(args.Errorf(0, err).Bytes(xs.Binder))
		} else if len(reply) != 1 {
			return c.ctx.Write(args.Errorf(-32603, "handle reply error").Bytes(xs.Binder))
		} else {
			return c.ctx.Write(reply[0].Bytes(xs.Binder))
		}
	} else {
		return c.ctx.Write(jsonrpc.NewError(-32600, "args error"))
	}
}

// handle services入口
func (xs *XServer) handle(sc *server.Context, node *registry.Node) error {
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
	reply, err := handler.handle(node, c)
	if err != nil {
		return err
	}
	return handler.Serialize(c, reply)
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
		xs.Server.AddHandler(node.Service.Name(), node.Name(), xs.rpcxHandle(node))
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
