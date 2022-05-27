package cosrpc

import (
	"github.com/smallnest/rpcx/server"
	"reflect"
)

type RegistryCaller func(c *server.Context, pr reflect.Value, fn reflect.Value) (interface{}, error)

type RegistrySerialize func(c *server.Context, reply interface{}) error

type RegistryInterface interface {
	Caller(c *server.Context, fn reflect.Value) interface{}
}

type RegistryHandler struct {
	Caller    RegistryCaller    //消息调用
	Metadata  []string          //metadata
	Serialize RegistrySerialize //消息序列化封装
}

func (this *RegistryHandler) Copy(src *RegistryHandler) {
	if src.Caller != nil {
		this.Caller = src.Caller
	}
	if src.Serialize != nil {
		this.Serialize = src.Serialize
	}
	if len(src.Metadata) > 0 {
		this.Metadata = append(this.Metadata, src.Metadata...)
	}
}

func (this *RegistryHandler) Use(src interface{}) {
	if v, ok := src.(*RegistryHandler); ok {
		this.Copy(v)
	}
	if v, ok := src.(RegistryCaller); ok {
		this.Caller = v
	}
	if v, ok := src.(RegistrySerialize); ok {
		this.Serialize = v
	}
	if v, ok := src.(string); ok {
		this.Metadata = append(this.Metadata, v)
	}
}
