package xserver

import (
	"github.com/hwcer/registry"
	"github.com/smallnest/rpcx/server"
)

type RegistryFilter func(s *registry.Service, node *registry.Node) bool
type RegistryCaller func(c *server.Context, node *registry.Node) (interface{}, error)
type RegistryMetadata func() string
type RegistrySerialize func(c *server.Context, reply interface{}) error

type registryInterface interface {
	Caller(c *server.Context, node *registry.Node) interface{}
}
type registryFilterHandle interface {
	Filter(s *registry.Service, node *registry.Node) bool
}
type registryCallerHandle interface {
	Caller(c *server.Context, node *registry.Node) (interface{}, error)
}
type registryMetadataHandle interface {
	Metadata() string
}
type registrySerializeHandle interface {
	Serialize(c *server.Context, reply interface{}) error
}

type Service struct {
	Filter    func(s *registry.Service, node *registry.Node) bool               // 接口过滤
	Caller    func(c *server.Context, node *registry.Node) (interface{}, error) //消息调用
	Metadata  []func() string                                                   //获取metadata
	Serialize func(c *server.Context, reply interface{}) error                  //消息序列化封装
}

func (this *Service) Use(src interface{}) {
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
