package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
)

var Manage = clients{}

// Discovery 注册中心服务发现,点对点或者点对多时无需设置

type clients struct {
	dict  map[string]*Client
	mutex sync.Mutex
}

func init() {
	Manage.dict = make(map[string]*Client)
	cosgo.On(cosgo.EventTypClosing, Manage.close)
	cosgo.On(cosgo.EventTypLoaded, Manage.reload)
	cosgo.On(cosgo.EventTypReload, Manage.reload)
}

// addServicePath 观察服务器信息
func (xc *clients) addServicePath(servicePath string, selector any) (c *Client, err error) {
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failover
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	err = c.start()
	return
}

func (xc *clients) close() (err error) {
	for _, c := range xc.dict {
		if err = c.close(); err != nil {
			return
		}
	}
	return
}
func (xc *clients) reload() (err error) {
	cs := make(map[string]*Client)
	for k, c := range xc.dict {
		cs[k] = c
	}
	var c *Client
	for name, value := range cosrpc.Service {
		s := xc.selector(name, value)
		if s == nil {
			return values.Errorf(0, "Service config error:%v %v", name, value)
		}
		if c, err = xc.addServicePath(name, s); err == nil {
			cs[c.ServicePath] = c
		} else {
			return
		}
	}
	Manage.dict = cs
	return
}

func (xc *clients) Has(servicePath string) bool {
	_, ok := xc.dict[servicePath]
	return ok
}

func (xc *clients) Get(servicePath string) (c client.XClient) {
	var err error
	if cs := xc.dict[servicePath]; cs != nil {
		c = cs.client
	} else if cs, err = xc.load(servicePath, cosrpc.SelectorTypeDiscovery); err == nil {
		c = cs.client
	} else {
		logger.Error(err)
	}
	return
}
func (xc *clients) Size() int {
	return len(xc.dict)
}

//func (xc *clients) Client(servicePath string) (c client.XClient, err error) {
//	if cs := xc.dict[servicePath]; cs != nil {
//		c = cs.client
//	} else if cs, err = xc.load(servicePath, SelectorTypeDiscovery); err == nil {
//		c = cs.client
//	}
//	return
//}

func (xc *clients) WithTimeout(req, res map[string]string) (context.Context, context.CancelFunc) {
	ctx, cancel := scc.WithTimeout(cosrpc.Timeout())
	if req != nil {
		ctx = context.WithValue(ctx, share.ReqMetaDataKey, req)
	}
	if res != nil {
		ctx = context.WithValue(ctx, share.ResMetaDataKey, res)
	}
	return ctx, cancel
}

func (xc *clients) Call(ctx context.Context, servicePath, serviceMethod string, args, reply any) error {
	c := xc.Get(servicePath)
	if c == nil {
		return fmt.Errorf("can not found any client:%s", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = scc.WithTimeout(cosrpc.Timeout())
		defer cancel()
	}
	serviceMethod = registry.Join(serviceMethod)
	return c.Call(ctx, serviceMethod, args, reply)
}

func (xc *clients) Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	c := xc.Get(servicePath)
	if c == nil {
		return fmt.Errorf("can not found any client:%v", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = xc.WithTimeout(nil, nil)
		defer cancel()
	}
	serviceMethod = registry.Join(serviceMethod)
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xc.Binder(ctx, binder.ContentTypeModReq).Marshal(args)
	}
	if err != nil {
		return
	}
	if err = c.Broadcast(ctx, serviceMethod, data, reply); err != nil {
		logger.Debug("Broadcast error:%v", err)
	}
	return
}

// XCall 使用默认的message发起请求
func (xc *clients) XCall(ctx context.Context, servicePath, serviceMethod string, args any, reply any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	if reply != nil && reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return errors.New("client.call reply must pointer")
	}
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xc.Binder(ctx, binder.ContentTypeModReq).Marshal(args)
	}
	if err != nil {
		return err
	}
	if _, ok := reply.(*[]byte); ok {
		if err = xc.Call(ctx, servicePath, serviceMethod, data, reply); err != nil {
			return err
		} else {
			return nil
		}
	}
	v := make([]byte, 0)
	if err = xc.Call(ctx, servicePath, serviceMethod, data, &v); err != nil {
		return err
	}
	if len(v) == 0 {
		return nil
	}
	msg := &values.Request{}
	if err = xc.Binder(ctx, binder.ContentTypeModReq).Unmarshal(v, msg); err != nil {
		return err
	}
	if reply != nil {
		err = msg.Unmarshal(reply)
	} else if msg.Code != 0 {
		err = msg
	}
	return err
}

func (xc *clients) Binder(ctx context.Context, mod binder.ContentTypeMod) (r binder.Binder) {
	return cosrpc.GetBinderFromContext(ctx, mod)
}

// Async 异步
func (xc *clients) Async(ctx context.Context, servicePath, serviceMethod string, args any) (done *Caller, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		if err != nil {
			logger.Debug("cosrpc Async err:%v\n%v", err, string(debug.Stack()))
		}
	}()
	c := xc.Get(servicePath)
	if c == nil {
		return nil, fmt.Errorf("can not found any client:%v", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = scc.WithTimeout(cosrpc.Timeout())
		defer cancel()
	}
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xc.Binder(ctx, binder.ContentTypeModReq).Marshal(args)
	}
	if err != nil {
		return nil, err
	}
	serviceMethod = registry.Join(serviceMethod)
	return c.Go(ctx, serviceMethod, data, nil, nil)
}

func (xc *clients) CallWithMetadata(req, res map[string]string, servicePath, serviceMethod string, args, reply any) (err error) {
	ctx, cancel := scc.WithTimeout(cosrpc.Timeout())
	defer cancel()
	if req != nil {
		ctx = context.WithValue(ctx, share.ReqMetaDataKey, req)
	}
	if res != nil {
		ctx = context.WithValue(ctx, share.ResMetaDataKey, res)
	}
	return xc.XCall(ctx, servicePath, serviceMethod, args, reply)
}

// load 在运行时创建
func (xc *clients) load(name string, selector any) (c *Client, err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	if c = xc.dict[name]; c != nil {
		return c, nil
	}
	cs := make(map[string]*Client)
	for k, v := range xc.dict {
		cs[k] = v
	}
	var s any
	switch v := selector.(type) {
	case string:
		s = xc.selector(name, v)
	default:
		s = selector
	}
	if c, err = xc.addServicePath(name, s); err == nil {
		cs[c.ServicePath] = c
	} else {
		return
	}
	xc.dict = cs
	return
}

func (xc *clients) selector(k, v string) (r any) {
	if s := strings.ToLower(v); s == cosrpc.SelectorTypeDiscovery {
		if r = cosrpc.Selector.Get(k); r == nil {
			r = selectorDefault
		}
	} else if s == cosrpc.SelectorTypeLocal {
		r = cosrpc.Address().String()
	} else if strings.Contains(v, ",") {
		r = strings.Split(v, ",")
	} else {
		r = v
	}
	return
}
