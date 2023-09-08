package cosrpc

import (
	"github.com/hwcer/cosrpc/jsonrpc"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
	"runtime/debug"
)

type jsonrpcHandle func(c *Context) Promise

type Promise interface {
	Caller(args *jsonrpc.Args) (*jsonrpc.Reply, error) //逐条调用
	Release()                                          //释放资源
}

func (xs *XServer) Jsonrpc(servicePath, serviceMethod string, handle jsonrpcHandle) {
	xs.jsonrpc = handle
	xs.Server.AddHandler(servicePath, serviceMethod, xs.jsonHandle)
}

// jsonHandle jsonrpc handle
func (xs *XServer) jsonHandle(sc *server.Context) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	c := NewContext(sc, xs.Binder)
	promise := xs.jsonrpc(c)
	defer promise.Release()

	b := c.Bytes()
	var err error
	if string(b[0:1]) == "[" {
		var args []*jsonrpc.Args
		var reply []*jsonrpc.Reply
		if err = xs.Binder.Unmarshal(b, &args); err != nil {
			return c.ctx.Write(jsonrpc.NewError(-32700, err).Bytes(xs.Binder))
		} else if len(args) == 0 {
			return c.ctx.Write(jsonrpc.NewError(-32600, "Invalid Request").Bytes(xs.Binder))
		} else {
			for _, arg := range args {
				if v, e := promise.Caller(arg); e != nil {
					reply = append(reply, arg.Errorf(0, e))
				} else {
					reply = append(reply, v)
				}
			}
			var data []byte
			if data, err = xs.Binder.Marshal(reply); err != nil {
				return c.ctx.Write(jsonrpc.NewError(-32603, "marshal reply error").Bytes(xs.Binder))
			} else {
				return c.ctx.Write(data)
			}
		}
	} else if string(b[0:1]) == "{" {
		args := &jsonrpc.Args{}
		var reply *jsonrpc.Reply
		if err = xs.Binder.Unmarshal(b, args); err != nil {
			return c.ctx.Write(jsonrpc.NewError(-32700, err).Bytes(xs.Binder))
		} else if reply, err = promise.Caller(args); err != nil {
			return c.ctx.Write(args.Errorf(0, err).Bytes(xs.Binder))
		} else {
			return c.ctx.Write(reply.Bytes(xs.Binder))
		}
	} else {
		return c.ctx.Write(jsonrpc.NewError(-32600, "args error"))
	}
}
