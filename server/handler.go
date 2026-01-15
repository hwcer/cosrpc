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

// HandlerFilter 定义服务过滤器
// 用于判断节点是否可以被调用
type HandlerFilter func(node *registry.Node) bool

// HandlerCaller 定义服务调用器
// 用于处理 RPC 请求并返回结果
type HandlerCaller func(node *registry.Node, c *cosrpc.Context) (interface{}, error)

// HandlerMetadata 定义服务元数据提供者
// 用于获取服务的元数据
type HandlerMetadata func() string

// HandlerMiddleware 定义服务中间件
// 用于处理请求前的逻辑，如认证、日志等
type HandlerMiddleware func(*cosrpc.Context) error

// HandlerSerialize 定义服务序列化器
// 用于序列化响应数据
type HandlerSerialize func(c *cosrpc.Context, reply interface{}) ([]byte, error)

// handleCaller 定义内部调用接口
// 用于统一处理不同类型的调用
type handleCaller interface {
	Caller(node *registry.Node, c *cosrpc.Context) interface{}
}

// Handler 是 cosrpc 服务器的处理器
// 支持多种处理器类型，如调用器、过滤器、元数据、中间件和序列化器
type Handler struct {
	caller     HandlerCaller     // 服务调用器
	filter     HandlerFilter     // 服务过滤器
	metadata   []HandlerMetadata // 服务元数据提供者
	middleware []HandlerMiddleware // 服务中间件
	serialize  HandlerSerialize  // 服务序列化器
}

// Use 应用一个处理器
// 根据处理器的类型，将其添加到对应的处理器列表中
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

// Filter 过滤节点
// 1. 如果有自定义过滤器，使用自定义过滤器
// 2. 否则，根据节点类型进行默认过滤
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

// Metadata 获取服务元数据
// 调用所有元数据提供者，将结果拼接成一个字符串
func (this *Handler) Metadata() string {
	var arr []string
	for _, f := range this.metadata {
		arr = append(arr, f())
	}
	return strings.Join(arr, "&")
}

// Caller 处理 RPC 请求
// 1. 执行所有中间件
// 2. 如果有自定义调用器，使用自定义调用器
// 3. 否则，根据节点类型进行默认调用
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

// Marshal 序列化响应数据
// 1. 如果有自定义序列化器，使用自定义序列化器
// 2. 否则，根据响应类型进行默认序列化
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
