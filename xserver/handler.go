package xserver

import (
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/message"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"github.com/smallnest/rpcx/server"
	"runtime/debug"
)

type handleCaller interface {
	Caller(node *registry.Node, c *Context) interface{}
}

type Handler struct {
}

func (this *Handler) Filter(_ *registry.Service, node *registry.Node) bool {
	if node.IsFunc() {
		_, ok := node.Method().(func(*Context) interface{})
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
	return true
}

func (this *Handler) Caller(c *server.Context, node *registry.Node) (reply interface{}, err error) {
	p := pool.Get().(*Context)
	defer func() {
		p.release()
		pool.Put(p)
		if v := recover(); v != nil {
			if cosgo.Debug() {
				reply = message.Errorf(500, v)
			} else {
				reply = message.Errorf(500, "server recover error")
			}
			logger.Info("rpc server recover error:%v\n%v", v, string(debug.Stack()))
		}
	}()
	err = p.reset(c)
	if err != nil {
		return nil, err
	}

	var v interface{}
	if node.IsFunc() {
		m := node.Method().(func(*Context) interface{})
		v = m(p)
	} else if s, ok := node.Value().Interface().(handleCaller); ok {
		v = s.Caller(node, p)
	} else {
		r := node.Call(p)
		v = r[0].Interface()
	}
	return message.Parse(v), nil
}
