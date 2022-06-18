package cosrpc

import (
	"github.com/hwcer/registry"
	"github.com/smallnest/rpcx/server"
	"reflect"
)

type RegistryFilter interface {
	Filter(s *registry.Service, pr, fn reflect.Value) bool
}

type RegistryCaller interface {
	Caller(c *server.Context, pr reflect.Value, fn reflect.Value) (interface{}, error)
}
type RegistrySerialize interface {
	Serialize(c *server.Context, reply interface{}) error
}

type RegistryMetadata interface {
	Metadata() string
}

type RegistryInterface interface {
	Caller(c *server.Context, fn reflect.Value) interface{}
}

type RegistryHandler struct {
	Filter    func(s *registry.Service, pr, fn reflect.Value) bool                             // 接口过滤
	Caller    func(c *server.Context, pr reflect.Value, fn reflect.Value) (interface{}, error) //消息调用
	Metadata  []func() string                                                                  //获取metadata
	Serialize func(c *server.Context, reply interface{}) error                                 //消息序列化封装
}

func (this *RegistryHandler) Copy(src *RegistryHandler) {
	if src.Filter != nil {
		this.Filter = src.Filter
	}
	if src.Caller != nil {
		this.Caller = src.Caller
	}
	if src.Serialize != nil {
		this.Serialize = src.Serialize
	}
	if src.Metadata != nil {
		this.Metadata = append(this.Metadata, src.Metadata...)
	}
}

func (this *RegistryHandler) Use(src interface{}) {
	if v, ok := src.(RegistryFilter); ok {
		this.Filter = v.Filter
	}
	if v, ok := src.(RegistryCaller); ok {
		this.Caller = v.Caller
	}
	if v, ok := src.(RegistrySerialize); ok {
		this.Serialize = v.Serialize
	}
	if v, ok := src.(RegistryMetadata); ok {
		this.Metadata = append(this.Metadata, v.Metadata)
	}
}
