package xshare

import (
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"reflect"
	"strings"
)

type HandlerFilter func(node *registry.Node) bool
type HandlerCaller func(node *registry.Node, c *Context) (interface{}, error)
type HandlerMetadata func() string
type HandlerSerialize func(c *Context, reply interface{}) ([]byte, error)

type handleCaller interface {
	Caller(node *registry.Node, c *Context) interface{}
}

type Handler struct {
	caller    HandlerCaller
	filter    HandlerFilter
	metadata  []HandlerMetadata
	serialize HandlerSerialize
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
	if v, ok := src.(HandlerSerialize); ok {
		this.serialize = v
	}
}

func (this *Handler) Filter(node *registry.Node) bool {
	if this.filter != nil {
		return this.filter(node)
	}
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

func (this *Handler) Metadata() string {
	var arr []string
	for _, f := range this.metadata {
		arr = append(arr, f())
	}
	return strings.Join(arr, "&")
}

//func (this *Handler) bytes(i any, bind binder.Interface) []byte {
//	v := values.NewMessage(i)
//	r, err := bind.Marshal(this)
//	if err != nil {
//		v.Format(0, err)
//		r, _ = bind.Marshal(this)
//	}
//	return r
//}

func (this *Handler) Caller(node *registry.Node, c *Context) (reply interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			reply = values.Errorf(500, "server recover error")
			logger.Error(e)
		}
	}()
	if this.caller != nil {
		return this.caller(node, c)
	}
	if node.IsFunc() {
		m := node.Method().(func(*Context) interface{})
		reply = m(c)
	} else if s, ok := node.Binder().(handleCaller); ok {
		reply = s.Caller(node, c)
	} else {
		r := node.Call(c)
		reply = r[0].Interface()
	}
	return
}
func (this *Handler) Marshal(c *Context, reply interface{}) (data []byte, err error) {
	if this.serialize != nil {
		return this.serialize(c, reply)
	}
	switch v := reply.(type) {
	case []byte:
		data = v
	case *[]byte:
		data = *v
	default:
		data, err = c.Binder.Marshal(values.Parse(reply))
	}
	return
}
