package jsonrpc

import "errors"

type ctx interface {
	Get(key any) any
	SetValue(key, val any)
	Payload() []byte
	Metadata() map[string]string
	ServicePath() string
	ServiceMethod() string
	Write(reply any) error
}

func New(ctx ctx, args *Args) *Context {
	return &Context{ctx: ctx, args: args}
}

type Context struct {
	ctx
	args *Args
}

func (this *Context) Payload() []byte {
	return this.args.Params
}
func (this *Context) ServiceMethod() string {
	return this.args.Method
}
func (this *Context) Write(_ any) error {
	return errors.New("jsonrpc prohibit this operation")
}
