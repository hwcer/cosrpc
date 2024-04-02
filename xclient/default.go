package xclient

import (
	"context"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/hwcer/registry"
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

// CallWithServerId 通过特定服务器ID发消息
func CallWithServerId(ctx context.Context, sid int32, servicePath, serviceMethod string, args, reply any) (err error) {
	metadata := make(xshare.Metadata)
	metadata.SetServerId(sid)
	return Default.CallWithMetadata(metadata.Json(), nil, servicePath, registry.Join(serviceMethod), args, reply)
}

// CallWithAddress 通过服务器地址发消息
func CallWithAddress(address string, servicePath, serviceMethod string, args, reply any) (err error) {
	metadata := make(xshare.Metadata)
	metadata.SetAddress(address)
	return Default.CallWithMetadata(metadata.Json(), nil, servicePath, registry.Join(serviceMethod), args, reply)
}

// CallWithMetadata 自定义metadata
func CallWithMetadata(req, res xshare.Metadata, servicePath, serviceMethod string, args, reply any) (err error) {
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
