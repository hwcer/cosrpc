package cosrpc

import (
	"encoding/json"
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/message"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"reflect"
	"runtime/debug"
	"strings"
)

type HandlerCaller func(node *registry.Node, c *Context) (interface{}, error)
type HandlerMetadata func() string
type HandlerSerialize func(c *Context, reply interface{}) error

type handleCaller interface {
	Caller(node *registry.Node, c *Context) interface{}
}

type Handler struct {
	caller    HandlerCaller
	metadata  []HandlerMetadata
	serialize HandlerSerialize
}

func (this *Handler) Use(src interface{}) {
	if v, ok := src.(HandlerCaller); ok {
		this.caller = v
	}
	if v, ok := src.(HandlerMetadata); ok {
		this.metadata = append(this.metadata, v)
	}
	if v, ok := src.(HandlerSerialize); ok {
		this.serialize = v
	}
}

func (this *Handler) Filter(node *registry.Node) bool {
	if node.IsFunc() {
		_, ok := node.Method().(func(*Context) interface{})
		return ok
	} else if node.IsMethod() {
		t := node.Value().Type()
		if t.NumIn() != 2 || t.NumOut() != 1 {
			return false
		}
		return true
	} else {
		if _, ok := node.Binder().(handleCaller); !ok {
			v := reflect.Indirect(reflect.ValueOf(node.Binder()))
			logger.Debug("[%v]未正确实现Caller方法,会影响程序性能", v.Type().String())
		}
		return true
	}
}

func (this *Handler) Caller(node *registry.Node, c *Context) (reply interface{}, err error) {
	defer func() {
		if v := recover(); v != nil {
			if cosgo.Debug() {
				reply = message.Errorf(500, v)
			} else {
				reply = message.Errorf(500, "server recover error")
			}
			logger.Info("rpc server recover error:%v\n%v", v, string(debug.Stack()))
		}
	}()
	if this.caller != nil {
		return this.caller(node, c)
	}
	if node.IsFunc() {
		m := node.Method().(func(*Context) interface{})
		reply = m(c)
	} else if s, ok := node.Method().(handleCaller); ok {
		reply = s.Caller(node, c)
	} else {
		r := node.Call(c)
		reply = r[0].Interface()
	}
	return
}

func (this *Handler) Metadata() string {
	var arr []string
	for _, f := range this.metadata {
		arr = append(arr, f())
	}
	return strings.Join(arr, "&")
}

func (this *Handler) Serialize(c *Context, reply interface{}) (err error) {
	if this.serialize != nil {
		return this.serialize(c, reply)
	}
	b, ok := reply.([]byte)
	if ok {
		return c.Write(b)
	}
	b, err = json.Marshal(message.Parse(reply))
	if err != nil {
		return err
	}
	return c.Write(b)
}
