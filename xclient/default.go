package xclient

import (
	"context"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/smallnest/rpcx/client"
	"time"
)

var Default = New(nil)

func ping(c *xshare.Context) interface{} {
	return time.Now().Unix()
}

func Service(name string, handler ...interface{}) *registry.Service {
	service := Default.Service(name, handler...)
	_ = service.Register(ping)
	return service
}

func Call(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	return Default.Call(ctx, servicePath, registry.Join(serviceMethod), args, reply)
}
func XCall(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	return Default.XCall(ctx, servicePath, registry.Join(serviceMethod), args, reply)
}

// Async 异步调用,仅仅调用无返回值
func Async(ctx context.Context, servicePath, serviceMethod string, args any) (call *client.Call, err error) {
	return Default.Async(ctx, servicePath, registry.Join(serviceMethod), args)
}

// CallWithMetadata 自定义metadata
func CallWithMetadata(req, res map[string]string, servicePath, serviceMethod string, args, reply any) (err error) {
	return Default.CallWithMetadata(req, res, servicePath, registry.Join(serviceMethod), args, reply)
}

func Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	return Default.Broadcast(ctx, servicePath, serviceMethod, args, reply)
}

func Start() (err error) {
	return Default.Start()
}

func Close() (err error) {
	return Default.Close()
}
