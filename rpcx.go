package cosrpc

import (
	"context"
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosrpc/share"
	"github.com/hwcer/registry"
	rpcxShare "github.com/smallnest/rpcx/share"
	"strconv"
	"time"
)

func ping(c *share.Context) interface{} {
	return time.Now().Unix()
}

func rpcContext() (context.Context, context.CancelFunc) {
	if cosgo.Debug() {
		return context.WithCancel(context.Background())
	}
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// Watch 默认注册RPC客户端方法
func Watch(servicePath ...string) {
	Client.Watch(servicePath...)
}

func Service(name string, handler ...interface{}) *registry.Service {
	service := Server.Service(name, handler...)
	_ = service.Register(ping)
	return service
}

// Async 异步调用,仅仅调用无返回值
func Async(servicePath, serviceMethod string, args interface{}, metadata map[string]string) (err error) {
	ctx, cancel := rpcContext()
	defer cancel()
	if metadata != nil && len(metadata) > 0 {
		ctx = context.WithValue(ctx, rpcxShare.ReqMetaDataKey, metadata)
	}
	return Client.Async(ctx, servicePath, registry.Join(serviceMethod), args)
}

func Call(servicePath, serviceMethod string, args, reply interface{}) (err error) {
	ctx, cancel := rpcContext()
	defer cancel()
	return Client.XCall(ctx, servicePath, registry.Join(serviceMethod), args, reply)
}

// CallWithPlayer 给特定用户发消息
//func CallWithPlayer(uid string, servicePath, serviceMethod string, args, reply interface{}) (err error) {
//	ctx, cancel := rpcContext()
//	defer cancel()
//	metadata := make(map[string]string)
//	metadata[MetadataUid] = uid
//	sid, err := UUID.ServerId(uid)
//	if err != nil {
//		return err
//	}
//	metadata[MetadataRpcServerId] = strconv.Itoa(int(sid))
//	ctx = context.WithValue(ctx, share.ReqMetaDataKey, metadata)
//	return Client.XCall(ctx, servicePath, PrivateServiceMethod(serviceMethod), args, reply)
//}

// CallWithServerId 通过特定服务器ID发消息
func CallWithServerId(sid int32, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	ctx, cancel := rpcContext()
	defer cancel()
	metadata := make(map[string]string)
	metadata[MetadataRpcServerId] = strconv.Itoa(int(sid))
	ctx = context.WithValue(ctx, rpcxShare.ReqMetaDataKey, metadata)
	return Client.XCall(ctx, servicePath, registry.Join(serviceMethod), args, reply)
}

// CallWithAddress 通过服务器地址发消息
func CallWithAddress(address string, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	ctx, cancel := rpcContext()
	defer cancel()
	metadata := make(map[string]string)
	metadata[MetadataRpcAddress] = RpcAddressFormat(address)
	ctx = context.WithValue(ctx, rpcxShare.ReqMetaDataKey, metadata)
	return Client.XCall(ctx, servicePath, registry.Join(serviceMethod), args, reply)
}

// CallWithMetadata 自定义metadata
func CallWithMetadata(req, res map[string]string, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	ctx, cancel := rpcContext()
	defer cancel()
	if req != nil {
		ctx = context.WithValue(ctx, rpcxShare.ReqMetaDataKey, req)
	}
	if res != nil {
		ctx = context.WithValue(ctx, rpcxShare.ResMetaDataKey, res)
	}
	return Client.XCall(ctx, servicePath, registry.Join(serviceMethod), args, reply)
}

//func Broadcast(servicePath, serviceMethod string, args, reply interface{}) (err error) {
//	return rpcx.Client.Broadcast(servicePath, rpcx.Server.Clean(serviceMethod), args, reply)
//}
