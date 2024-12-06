package inprocess

import (
	"bytes"
	"errors"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc/xserver"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/smallnest/rpcx/share"
	"reflect"
	"runtime/debug"
)

func CallWithMetadata(req, res map[string]string, servicePath, serviceMethod string, args, reply any) (err error) {
	meta := make(map[any]any)
	if req != nil {
		meta[share.ReqMetaDataKey] = req
	}
	if res != nil {
		meta[share.ResMetaDataKey] = res
	}
	return Call(meta, servicePath, serviceMethod, args, reply)
}

// Call 使用默认的message发起请求
func Call(meta map[any]any, servicePath, serviceMethod string, args any, reply any) error {
	if reply != nil && reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return errors.New("client.call reply must pointer")
	}
	var err error
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xshare.Binder.Marshal(args)
	}
	if err != nil {
		return err
	}
	if v, ok := reply.(*[]byte); ok {
		if err = caller(meta, servicePath, serviceMethod, data, v); err != nil {
			return err
		} else {
			return nil
		}
	}
	v := make([]byte, 0)
	if err = caller(meta, servicePath, serviceMethod, data, &v); err != nil {
		return err
	}
	msg := &values.Message{}
	if err = xshare.Binder.Unmarshal(v, msg); err != nil {
		return err
	}
	if reply != nil {
		err = msg.Unmarshal(reply)
	} else if msg.Code != 0 {
		err = msg
	}
	return err
}

func caller(meta map[any]any, servicePath, serviceMethod string, data []byte, reply *[]byte) (err error) {
	node, ok := xserver.Default.Registry.Match(servicePath, serviceMethod)
	if !ok {
		return errors.New("services not found: " + serviceMethod)
	}
	req := &Request{}
	req.ServicePath = servicePath
	req.ServiceMethod = serviceMethod
	req.Payload = data
	sc := &Context{req: req, meta: meta}
	sc.reply = bytes.Buffer{}
	if err = handle(sc, node); err == nil {
		*reply = sc.reply.Bytes()
	}
	return err
}

// caller services入口
func handle(sc *Context, node *registry.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	handler, ok := node.Service.Handler.(*xshare.Handler)
	if !ok {
		return errors.New("handler unknown")
	}
	c := xshare.NewContext(sc)
	var reply any
	reply, err = handler.Caller(node, c)
	if err != nil {
		return
	}
	var data []byte
	if data, err = handler.Marshal(c, reply); err == nil {
		return c.Write(data)
	}
	return
}
