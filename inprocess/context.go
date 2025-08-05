package inprocess

import (
	"bytes"
	"fmt"
	"github.com/smallnest/rpcx/share"
)

//type cxt interface {
//	Get(key any) any
//	SetValue(key, val any)
//	Payload() []byte
//	Metadata() map[string]string
//	ServicePath() string
//	ServiceMethod() string
//	Write(reply any) error
//}

// /
type Context struct {
	req   *Request
	meta  map[any]any
	reply bytes.Buffer
}

// Get returns value for key.
func (ctx *Context) Get(key interface{}) interface{} {
	return ctx.meta[key]
}

// SetValue sets the kv pair.
func (ctx *Context) SetValue(key, val interface{}) {
	if key == nil || val == nil {
		return
	}
	ctx.meta[key] = val
}

// DeleteKey delete the kv pair by key.
func (ctx *Context) DeleteKey(key interface{}) {
	if ctx.meta == nil || key == nil {
		return
	}
	delete(ctx.meta, key)
}

// Payload returns the  payload.
func (ctx *Context) Payload() []byte {
	return ctx.req.Payload
}

// Metadata returns the metadata.
func (ctx *Context) Metadata() map[string]string {
	if i, ok := ctx.meta[share.ReqMetaDataKey]; ok {
		return i.(map[string]string)
	}
	return map[string]string{}
}

func (ctx *Context) GetMetadata(name string) string {
	meta := ctx.Metadata()
	return meta[name]
}

// ServicePath returns the ServicePath.
func (ctx *Context) ServicePath() string {
	return ctx.req.ServicePath
}

// ServiceMethod returns the ServiceMethod.
func (ctx *Context) ServiceMethod() string {
	return ctx.req.ServiceMethod
}

// Bind parses the body data and stores the result to v.
func (ctx *Context) Bind(v interface{}) error {
	req := ctx.req
	if v != nil {
		codec := share.Codecs[req.SerializeType()]
		if codec == nil {
			return fmt.Errorf("can not find codec for %d", req.SerializeType())
		}

		err := codec.Decode(req.Payload, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx *Context) Write(v interface{}) (err error) {
	req := ctx.req
	codec := share.Codecs[req.SerializeType()]
	if codec == nil {
		return fmt.Errorf("can not find codec for %d", req.SerializeType())
	}
	var b []byte
	if b, err = codec.Encode(v); err == nil {
		_, err = ctx.reply.Write(b)
	}
	return
}
