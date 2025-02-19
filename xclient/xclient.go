package xclient

import (
	"context"
	"errors"
	"fmt"
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
	"time"
)

// Discovery 注册中心服务发现,点对点或者点对多时无需设置
type Discovery func() (client.ServiceDiscovery, error)

func New(ctx context.Context) *XClient {
	return &XClient{
		scc:       scc.New(ctx),
		clients:   &clients{},
		Registry:  registry.New(nil),
		Discovery: xshare.Discovery,
	}
}

type clients struct {
	dict map[string]*Client
}

func (c *clients) get(key string) *Client {
	return c.dict[key]
}
func (c *clients) set(key string, client *Client) {
	if c.dict == nil {
		c.dict = make(map[string]*Client)
	}
	c.dict[key] = client
}
func (c *clients) has(key string) bool {
	_, ok := c.dict[key]
	return ok
}

func (c *clients) copy(cs *clients) {
	for k, v := range cs.dict {
		c.set(k, v)
	}
}

type XClient struct {
	scc       *scc.SCC
	mutex     sync.Mutex
	started   bool
	clients   *clients
	message   chan *protocol.Message
	Registry  *registry.Registry
	Discovery Discovery
}

// addServicePath 观察服务器信息
func (xc *XClient) addServicePath(servicePath string, selector any) (c *Client, err error) {
	if c = xc.clients.get(servicePath); c != nil {
		err = c.Reload(selector)
		return
	}
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failover
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	cs := &clients{}
	cs.copy(xc.clients)
	cs.set(servicePath, c)
	xc.clients = cs
	err = c.Start(xc.Discovery, xc.message)
	return
}

func (xc *XClient) Start() (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	return xc.start()
}

func (xc *XClient) Close() (err error) {
	if !xc.scc.Cancel() {
		return
	}
	for _, c := range xc.clients.dict {
		if err = c.Close(); err != nil {
			return
		}
	}
	return xc.scc.Wait(time.Second)
}

func (xc *XClient) Reload() (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	if !xc.started {
		return xc.start()
	} else {
		return xc.reload()
	}
}

func (xc *XClient) Has(servicePath string) bool {
	return xc.clients.has(servicePath)
}

func (xc *XClient) Size() int {
	return len(xc.clients.dict)
}

func (xc *XClient) Client(servicePath string) client.XClient {
	if r := xc.clients.get(servicePath); r != nil {
		return r.client
	} else {
		return nil
	}

}

func (xc *XClient) Call(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	c := xc.Client(servicePath)
	if c == nil {
		return fmt.Errorf("can not found any server:%v", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = xc.scc.WithTimeout(xshare.Timeout())
		defer cancel()
	}
	return c.Call(ctx, serviceMethod, args, reply)
}

func (xc *XClient) Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply any) (err error) {
	c := xc.Client(servicePath)
	if c == nil {
		return fmt.Errorf("can not found any server:%v", servicePath)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = xc.scc.WithTimeout(xshare.Timeout())
		defer cancel()
	}
	return c.Broadcast(ctx, serviceMethod, args, reply)
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
		ctx, cancel = xc.scc.WithTimeout(xshare.Timeout())
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
	ctx, cancel := xc.scc.WithTimeout(xshare.Timeout())
	defer cancel()
	if req != nil {
		ctx = context.WithValue(ctx, share.ReqMetaDataKey, req)
	}
	if res != nil {
		ctx = context.WithValue(ctx, share.ResMetaDataKey, res)
	}
	return xc.XCall(ctx, servicePath, serviceMethod, args, reply)
}

// Service 注册服务处理 server 端推送消息
func (xc *XClient) Service(name string, handler ...interface{}) *registry.Service {
	service := xc.Registry.Service(name)
	if service.Handler == nil {
		service.Handler = &xshare.Handler{}
	}
	if h, ok := service.Handler.(*xshare.Handler); ok {
		for _, i := range handler {
			h.Use(i)
		}
	}
	return service
}

// 处理 server 端推送消息
func (xc *XClient) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-xc.message:
			if msg == nil {
				return
			} else {
				xc.handle(msg)
			}
		}
	}
}
func (xc *XClient) handle(msg *protocol.Message) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	//logger.Trace("XClient Message:%v", string(msg.Payload))
	node, ok := xc.Registry.Match(msg.ServicePath, msg.ServiceMethod)
	if !ok {
		logger.Debug("XClient handle not found,ServicePath:%v  ServiceMethod:%v", msg.ServicePath, msg.ServiceMethod)
		return
	}

	handler, ok := node.Service.Handler.(*xshare.Handler)
	if !ok {
		logger.Debug("XClient Service handler unknown")
		return
	}

	c := xshare.NewContext(&Context{Message: msg})
	if _, err := handler.Caller(node, c); err != nil {
		logger.Debug(err)
	}
}

func (xc *XClient) start() (err error) {
	if xc.started {
		return
	}
	xc.started = true

	if xc.Registry.Len() > 0 {
		xc.message = make(chan *protocol.Message, xshare.Options.ClientMessageChan)
		for i := 0; i < xshare.Options.ClientMessageWorker; i++ {
			xc.scc.CGO(xc.worker)
		}
	}
	if err = xc.reload(); err != nil {
		return
	}
	return
}

func (xc *XClient) reload() (err error) {
	for name, value := range xshare.Service {
		if selector := xc.selector(name, value); selector == nil {
			return values.Errorf(0, "Service config error:%v %v", name, value)
		} else if _, err = xc.addServicePath(name, selector); err != nil {
			return
		}
	}
	return
}

func (xc *XClient) selector(k, v string) (r any) {
	if s := strings.ToLower(v); s == xshare.SelectorTypeDiscovery {
		return xshare.NewSelector(k)
	} else if s == xshare.SelectorTypeLocal {
		return xshare.Address().String()
	} else if strings.Contains(v, ",") {
		r = strings.Split(v, ",")
	} else {
		r = v
	}
	return
}
