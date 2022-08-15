package cosrpc

import (
	"github.com/hwcer/registry"
	"github.com/smallnest/rpcx/server"
	"reflect"
)

type RegistryFilter func(s *registry.Service, pr, fn reflect.Value) bool
type RegistryCaller func(c *server.Context, pr reflect.Value, fn reflect.Value) (interface{}, error)
type RegistryMetadata func() string
type RegistrySerialize func(c *server.Context, reply interface{}) error

type registryInterface interface {
	Caller(c *server.Context, fn reflect.Value) interface{}
}
type registryFilterHandle interface {
	Filter(s *registry.Service, pr, fn reflect.Value) bool
}
type registryCallerHandle interface {
	Caller(c *server.Context, pr reflect.Value, fn reflect.Value) (interface{}, error)
}
type registryMetadataHandle interface {
	Metadata() string
}
type registrySerializeHandle interface {
	Serialize(c *server.Context, reply interface{}) error
}

type Registry struct {
	Filter    func(s *registry.Service, pr, fn reflect.Value) bool                             // 接口过滤
	Caller    func(c *server.Context, pr reflect.Value, fn reflect.Value) (interface{}, error) //消息调用
	Metadata  []func() string                                                                  //获取metadata
	Serialize func(c *server.Context, reply interface{}) error                                 //消息序列化封装
}

func (this *Registry) Copy(src *Registry) {
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

func (this *Registry) Use(src interface{}) {
	if v, ok := src.(RegistryFilter); ok {
		this.Filter = v
	}
	if v, ok := src.(RegistryCaller); ok {
		this.Caller = v
	}
	if v, ok := src.(RegistryMetadata); ok {
		this.Metadata = append(this.Metadata, v)
	}
	if v, ok := src.(RegistrySerialize); ok {
		this.Serialize = v
	}

	if v, ok := src.(registryFilterHandle); ok {
		this.Filter = v.Filter
	}
	if v, ok := src.(registryCallerHandle); ok {
		this.Caller = v.Caller
	}
	if v, ok := src.(registryMetadataHandle); ok {
		this.Metadata = append(this.Metadata, v.Metadata)
	}
	if v, ok := src.(registrySerializeHandle); ok {
		this.Serialize = v.Serialize
	}

}
