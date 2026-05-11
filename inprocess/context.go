package inprocess

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/smallnest/rpcx/share"
)

type Context struct {
	req   *Request
	meta  map[any]any
	reply any
}

// Get returns value for key.
func (ctx *Context) Get(key any) any {
	return ctx.meta[key]
}

// SetValue sets the kv pair.
func (ctx *Context) SetValue(key, val any) {
	if key == nil || val == nil {
		return
	}
	ctx.meta[key] = val
}

// DeleteKey delete the kv pair by key.
func (ctx *Context) DeleteKey(key any) {
	if ctx.meta == nil || key == nil {
		return
	}
	delete(ctx.meta, key)
}

// Payload 返回序列化后的二进制数据
// 优先返回已有的 Payload，否则从 Args 懒序列化并缓存
func (ctx *Context) Payload() []byte {
	if ctx.req.Payload == nil && ctx.req.Args != nil {
		ctx.req.Payload, _ = binder.Json.Marshal(ctx.req.Args)
	}
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

// Bind 优先从 Args 直接反序列化，回退到 Payload 字节流
func (ctx *Context) Bind(v any) error {
	if v == nil {
		return nil
	}
	if ctx.req.Args != nil {
		return Unmarshal(ctx.req.Args, v)
	}
	if len(ctx.req.Payload) > 0 {
		return binder.Json.Unmarshal(ctx.req.Payload, v)
	}
	return nil
}

func (ctx *Context) Write(v any) (err error) {
	ctx.reply = v
	return nil
}
