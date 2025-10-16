package server

import (
	"reflect"
	"strings"

	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc"
	"github.com/hwcer/logger"
)

type HandlerFilter func(node *registry.Node) bool
type HandlerCaller func(node *registry.Node, c *cosrpc.Context) (interface{}, error)
type HandlerMetadata func() string
type HandlerMiddleware func(*cosrpc.Context) error
type HandlerSerialize func(c *cosrpc.Context, reply interface{}) ([]byte, error)

type handleCaller interface {
	Caller(node *registry.Node, c *cosrpc.Context) interface{}
}

type Handler struct {
	caller     HandlerCaller
	filter     HandlerFilter
	metadata   []HandlerMetadata
	middleware []HandlerMiddleware
	serialize  HandlerSerialize
}

func (this *Handler) Use(src interface{}) {
	if v, ok := src.(HandlerCaller); ok {
		this.caller = v
	}
	if v, ok := src.(HandlerFilter); ok {
		this.filter = v
	}
	if v, ok := src.(HandlerMetadata); ok {
		this.metadata = append(this.metadata, v)
	}
	if v, ok := src.(HandlerMiddleware); ok {
		this.middleware = append(this.middleware, v)
	}
	if v, ok := src.(HandlerSerialize); ok {
		this.serialize = v
	}
}

func (this *Handler) Filter(node *registry.Node) bool {
	if this.filter != nil {
		return this.filter(node)
	}
	if node.IsFunc() {
		_, ok := node.Method().(func(*cosrpc.Context) interface{})
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

func (this *Handler) Metadata() string {
	var arr []string
	for _, f := range this.metadata {
		arr = append(arr, f())
	}
	return strings.Join(arr, "&")
}

func (this *Handler) Caller(node *registry.Node, c *cosrpc.Context) (reply interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = values.Errorf(500, "server recover error")
			logger.Error(e)
		}
	}()
	for _, m := range this.middleware {
		if err = m(c); err != nil {
			return
		}
	}
	if this.caller != nil {
		return this.caller(node, c)
	}
	if node.IsFunc() {
		m := node.Method().(func(*cosrpc.Context) interface{})
		reply = m(c)
	} else if s, ok := node.Binder().(handleCaller); ok {
		reply = s.Caller(node, c)
	} else {
		r := node.Call(c)
		reply = r[0].Interface()
	}
	return
}
func (this *Handler) Marshal(c *cosrpc.Context, reply any) (data []byte, err error) {
	if reply == nil {
		return
	}
	if this.serialize != nil {
		return this.serialize(c, reply)
	}
	switch v := reply.(type) {
	case []byte:
		data = v
	case *[]byte:
		data = *v
	default:
		data, err = c.Binder(binder.ContentTypeModRes).Marshal(values.Parse(reply))
	}
	return
}
