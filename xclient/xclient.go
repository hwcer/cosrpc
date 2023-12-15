package xclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc/share"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
)

// 注册中心服务发现,点对点或者点对多时无需设置
type RegistryDiscovery func() (client.ServiceDiscovery, error)

func NewXClient() *XClient {
	return &XClient{
		clients:  make(map[string]*Client),
		Binder:   binder.New(binder.MIMEJSON),
		Registry: registry.New(nil),
	}
}

type XClient struct {
	mutex     sync.Mutex
	started   bool
	clients   map[string]*Client
	message   chan *protocol.Message
	Binder    binder.Interface
	Registry  *registry.Registry
	Discovery RegistryDiscovery
}

// AddServicePath 观察服务器信息
func (xc *XClient) AddServicePath(servicePath string, selector any) (c *Client, err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	exist := xc.clients[servicePath]
	if exist != nil && share.Equal(exist.Selector, selector) {
		c = exist
		return
	}
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failover
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	if xc.started {
		if err = c.Start(xc.Discovery, xc.message); err != nil {
			return
		}
	}
	clients := map[string]*Client{}
	for k, v := range xc.clients {
		clients[k] = v
	}
	clients[servicePath] = c
	xc.clients = clients

	if exist != nil {
		//服务重定向，关闭旧的client
		time.AfterFunc(5*time.Second, func() {
			_ = exist.client.Close()
		})
	}
	return
}

func (xc *XClient) Start() (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	if xc.started {
		return
	}
	xc.started = true
	if xc.Registry.Len() > 0 {
		xc.message = make(chan *protocol.Message, 1024)
		for i := 0; i < 10; i++ {
			scc.CGO(xc.worker)
		}
	}
	for _, c := range xc.clients {
		if err = c.Start(xc.Discovery, xc.message); err != nil {
			return
		}
	}
	return
}

func (xc *XClient) Close() (err error) {
	for _, c := range xc.clients {
		if err = c.client.Close(); err != nil {
			return
		}
	}
	return
}

func (xc *XClient) Has(servicePath string) bool {
	if _, ok := xc.clients[servicePath]; ok {
		return true
	} else {
		return false
	}
}

func (xc *XClient) Size() int {
	return len(xc.clients)
}

func (xc *XClient) Client(servicePath string) client.XClient {
	if v, ok := xc.clients[servicePath]; ok {
		return v.client
	} else {
		return nil
	}
}

func (xc *XClient) Call(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c := xc.Client(servicePath); c != nil {
		return c.Call(ctx, serviceMethod, args, reply)
	} else {
		return fmt.Errorf("can not found any server:%v", servicePath)
	}
}

func (xc *XClient) Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c := xc.Client(servicePath); c != nil {
		return c.Broadcast(ctx, serviceMethod, args, reply)
	} else {
		return fmt.Errorf("服务不存在")
	}
}

// XCall 使用默认的message发起请求
func (xc *XClient) XCall(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	if reply != nil && reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return errors.New("client.call reply must pointer")
	}
	var err error
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xc.Binder.Marshal(args)
	}
	if err != nil {
		return err
	}
	if _, ok := reply.(*[]byte); ok {
		if err = xc.Call(ctx, servicePath, serviceMethod, data, reply); err != nil {
			//return ParseError(err)
			return err
		} else {
			return nil
		}
	}
	v := make([]byte, 0)
	err = xc.Call(ctx, servicePath, serviceMethod, data, &v)

	var msg *values.Message
	var isReplyMsg bool
	if reply == nil {
		msg = &values.Message{}
	} else if msg, isReplyMsg = reply.(*values.Message); !isReplyMsg {
		msg = &values.Message{}
	}
	if err == nil {
		err = xc.Binder.Unmarshal(v, msg)
	}
	if err != nil {
		msg.Format(0, err)
	}
	if isReplyMsg {
		return nil
	} else if msg.Code != 0 {
		return errors.New(msg.String())
	} else if reply != nil {
		return msg.Unmarshal(reply)
	} else {
		return nil
	}
}

// Async 异步
func (xc *XClient) Async(ctx context.Context, servicePath, serviceMethod string, args interface{}) (err error) {
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = xc.Binder.Marshal(args)
	}
	if err != nil {
		return err
	}
	if c := xc.Client(servicePath); c != nil {
		_, err = c.Go(ctx, serviceMethod, data, nil, nil)
	} else {
		err = client.ErrXClientNoServer
	}
	return err
}

func (xc *XClient) Service(name string, handler ...interface{}) *registry.Service {
	service := xc.Registry.Service(name)
	if service.Handler == nil {
		service.Handler = &share.Handler{}
	}
	if h, ok := service.Handler.(*share.Handler); ok {
		for _, i := range handler {
			h.Use(i)
		}
	}
	return service
}

// 处理 server 端推送消息
func (xc *XClient) worker(ctx context.Context) {
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
func (xc *XClient) handle(msg *protocol.Message) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	logger.Trace("XClient Message:%v", string(msg.Payload))
	node, ok := xc.Registry.Match(msg.ServicePath, msg.ServiceMethod)
	if !ok {
		logger.Debug("XClient handle not found,ServicePath:%v  ServiceMethod:%v", msg.ServicePath, msg.ServiceMethod)
		return
	}

	handler, ok := node.Service.Handler.(*share.Handler)
	if !ok {
		logger.Debug("XClient Service handler unknown")
		return
	}

	c := share.NewContext(&Context{Message: msg}, xc.Binder)
	if _, err := handler.Caller(node, c); err != nil {
		logger.Debug(err)
	}
}
