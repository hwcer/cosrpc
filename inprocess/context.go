package inprocess

import (
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/share"
)

// /
type Context struct {
	req   *Request
	meta  map[any]any
	reply any
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
	logger.Alert("Payload is nil in inprocess mode")
	return nil // inprocess 模式下，payload 为空
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
		return Unmarshal(req.Payload, v)
	}
	return nil
}

func (ctx *Context) Write(v interface{}) (err error) {
	ctx.reply = v
	return nil
}
