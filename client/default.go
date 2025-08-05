package client

import (
	"context"
	"github.com/hwcer/cosrpc"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/client"
	"reflect"
	"time"
)

type Caller = client.Call

type discovery func(ServicePath string) (client.ServiceDiscovery, error)

// selectorDefault 默认选择器
var selectorDefault any = client.RandomSelect
var discoveryDefault discovery

func SetSelector(s any) {
	switch s.(type) {
	case client.Selector, client.SelectMode:
		selectorDefault = s
	default:
		logger.Error("selector type error:%v", reflect.TypeOf(s).Kind())
	}
}

func SetDiscovery(d discovery) {
	discoveryDefault = d
}

func ping(c *cosrpc.Context) interface{} {
	return time.Now().Unix()
}

func Call(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	return Manage.Call(ctx, servicePath, serviceMethod, args, reply)
}
func XCall(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	return Manage.XCall(ctx, servicePath, serviceMethod, args, reply)
}

// Async 异步调用,仅仅调用无返回值
func Async(ctx context.Context, servicePath, serviceMethod string, args any) (call *Caller, err error) {
	return Manage.Async(ctx, servicePath, serviceMethod, args)
}

// CallWithMetadata 自定义metadata
func CallWithMetadata(req, res map[string]string, servicePath, serviceMethod string, args, reply any) (err error) {
	return Manage.CallWithMetadata(req, res, servicePath, serviceMethod, args, reply)
}
func Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	return Manage.Broadcast(ctx, servicePath, serviceMethod, args, reply)
}
func WithTimeout(req, res map[string]string) (context.Context, context.CancelFunc) {
	return Manage.WithTimeout(req, res)
}
