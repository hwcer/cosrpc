package xclient

import (
	"errors"
	"github.com/hwcer/cosgo/logger"
	"github.com/smallnest/rpcx/protocol"
)

type Context struct {
	*protocol.Message
}

func (ctx *Context) Get(key any) any {
	logger.Alert("rpcx client端不允许使用Get方法")
	return nil
}
func (ctx *Context) SetValue(key, val any) {
	logger.Alert("rpcx client端不允许使用SetValue方法")
}
func (ctx *Context) Payload() []byte {
	return ctx.Message.Payload
}
func (ctx *Context) Metadata() map[string]string {
	r := map[string]string{}
	for k, v := range ctx.Message.Metadata {
		r[k] = v
	}
	return r
}
func (ctx *Context) ServicePath() string {
	return ctx.Message.ServicePath
}
func (ctx *Context) ServiceMethod() string {
	return ctx.Message.ServiceMethod
}
func (ctx *Context) Write(reply any) error {
	return errors.New("rpcx client端无法使用Write回复消息")
}
