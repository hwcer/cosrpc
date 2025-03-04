package xclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
)

// Discovery 注册中心服务发现,点对点或者点对多时无需设置
type Discovery func(ServicePath string) (client.ServiceDiscovery, error)

func New() *XClient {
	return &XClient{}
}

type clients map[string]*Client

func (c clients) get(key string) *Client {
	return c[key]
}
func (c clients) set(key string, client *Client) {
	c[key] = client
}
func (c clients) has(key string) bool {
	_, ok := c[key]
	return ok
}

func (c clients) copy(cs clients) {
	for k, v := range cs {
		c.set(k, v)
	}
}

type XClient struct {
	mutex     sync.Mutex
	started   bool
	clients   clients
	Discovery Discovery
}

// addServicePath 观察服务器信息
func (xc *XClient) addServicePath(servicePath string, selector any) (c *Client, err error) {
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failover
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	err = c.Start(xc.Discovery)
	return
}

func (xc *XClient) Start(discovery Discovery) (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	if discovery != nil {
		xc.Discovery = discovery
	}
	if err = xc.start(); err == nil {
		cosgo.On(cosgo.EventTypClosing, xc.close)
	}
	return
}

func (xc *XClient) close() (err error) {
	for _, c := range xc.clients {
		if err = c.Close(); err != nil {
			return
		}
	}
	return nil
}

func (xc *XClient) Reload() (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	if !xc.started {
		return errors.New("XClient is not started")
	}
	return xc.reload()
}

func (xc *XClient) Has(servicePath string) bool {
	return xc.clients.has(servicePath)
}

func (xc *XClient) Size() int {
	return len(xc.clients)
}

func (xc *XClient) Client(servicePath string) client.XClient {
	if r := xc.clients.get(servicePath); r != nil {
		return r.client
	} else {
		return nil
	}

}

func (xc *XClient) WithTimeout(req, res map[string]string) (context.Context, context.CancelFunc) {
	ctx, cancel := scc.WithTimeout(xshare.Timeout())
	if req != nil {
		ctx = context.WithValue(ctx, share.ReqMetaDataKey, req)
	}
	if res != nil {
		ctx = context.WithValue(ctx, share.ResMetaDataKey, res)
	}
	return ctx, cancel
}

func (xc *XClient) Call(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	c := xc.Client(servicePath)
	if c == nil {
		return fmt.Errorf("can not found any server:%v", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = scc.WithTimeout(xshare.Timeout())
		defer cancel()
	}
	serviceMethod = registry.Join(serviceMethod)
	return c.Call(ctx, serviceMethod, args, reply)
}

func (xc *XClient) Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	c := xc.Client(servicePath)
	if c == nil {
		return fmt.Errorf("can not found any server:%v", servicePath)
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
		data, err = xc.Binder(ctx, xshare.BinderModReq).Marshal(args)
	}
	if err = c.Broadcast(ctx, serviceMethod, data, reply); err != nil {
		logger.Debug("Broadcast error:%v", err)
	}
	return
}

// XCall 使用默认的message发起请求
func (xc *XClient) XCall(ctx context.Context, servicePath, serviceMethod string, args any, reply any) (err error) {
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
		data, err = xc.Binder(ctx, xshare.BinderModReq).Marshal(args)
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
	msg := &values.Message{}
	if err = xc.Binder(ctx, xshare.BinderModRes).Unmarshal(v, msg); err != nil {
		return err
	}
	if reply != nil {
		err = msg.Unmarshal(reply)
	} else if msg.Code != 0 {
		err = msg
	}
	return err
}

func (xc *XClient) Binder(ctx context.Context, mod xshare.BinderMod) (r binder.Binder) {
	return xshare.GetBinderFromContext(ctx, mod)
}

// Async 异步
func (xc *XClient) Async(ctx context.Context, servicePath, serviceMethod string, args any) (done *client.Call, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
		if err != nil {
			logger.Debug("cosrpc Async err:%v\n%v", err, string(debug.Stack()))
		}
	}()
	c := xc.Client(servicePath)
	if c == nil {
		return nil, fmt.Errorf("can not found any server:%v", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = scc.WithTimeout(xshare.Timeout())
		defer cancel()
	}
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xc.Binder(ctx, xshare.BinderModReq).Marshal(args)
	}
	if err != nil {
		return nil, err
	}
	return c.Go(ctx, serviceMethod, data, nil, nil)
}

func (xc *XClient) CallWithMetadata(req, res map[string]string, servicePath, serviceMethod string, args, reply any) (err error) {
	ctx, cancel := scc.WithTimeout(xshare.Timeout())
	defer cancel()
	if req != nil {
		ctx = context.WithValue(ctx, share.ReqMetaDataKey, req)
	}
	if res != nil {
		ctx = context.WithValue(ctx, share.ResMetaDataKey, res)
	}
	return xc.XCall(ctx, servicePath, serviceMethod, args, reply)
}

func (xc *XClient) start() (err error) {
	if xc.started {
		return
	}
	xc.started = true
	if err = xc.reload(); err != nil {
		return
	}
	return
}

func (xc *XClient) reload() (err error) {
	cs := clients{}
	cs.copy(xc.clients)
	var c *Client
	for name, value := range xshare.Service {
		selector := xc.selector(name, value)
		if selector == nil {
			return values.Errorf(0, "Service config error:%v %v", name, value)
		}
		if c, err = xc.addServicePath(name, selector); err == nil {
			cs[c.ServicePath] = c
		} else {
			return
		}
	}
	xc.clients = cs
	return
}

func (xc *XClient) selector(k, v string) (r any) {
	if s := strings.ToLower(v); s == xshare.SelectorTypeDiscovery {
		if r = xshare.Selector.Get(k); r == nil {
			r = client.RandomSelect
		}
	} else if s == xshare.SelectorTypeLocal {
		r = xshare.Address().String()
	} else if strings.Contains(v, ",") {
		r = strings.Split(v, ",")
	} else {
		r = v
	}
	return
}
