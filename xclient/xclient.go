package xclient

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"github.com/hwcer/scc"
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
		clients:   make(map[string]*Client),
		Binder:    binder.New(binder.MIMEJSON),
		Registry:  registry.New(nil),
		Discovery: xshare.Discovery,
	}
}

type XClient struct {
	scc       *scc.SCC
	mutex     sync.Mutex
	start     bool
	clients   map[string]*Client
	message   chan *protocol.Message
	Binder    binder.Interface
	Registry  *registry.Registry
	Discovery Discovery
}

// addServicePath 观察服务器信息
func (xc *XClient) addServicePath(servicePath string, selector any) (c *Client, err error) {
	exist := xc.clients[servicePath]
	if exist != nil && xshare.Equal(exist.Selector, selector) {
		c = exist
		return
	}
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failover
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	clients := map[string]*Client{}
	for k, v := range xc.clients {
		clients[k] = v
	}
	clients[servicePath] = c
	xc.clients = clients
	if !xc.start {
		return
	}
	if exist != nil {
		time.AfterFunc(5*time.Second, func() {
			_ = exist.client.Close()
		})
	}
	err = c.Start(xc.Discovery, xc.message)
	return
}

func (xc *XClient) Start() (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	if xc.start {
		return
	}
	defer func() {
		xc.start = true
	}()

	if err = xc.reload(); err != nil {
		return
	}

	if xc.Registry.Len() > 0 {
		xc.message = make(chan *protocol.Message, 1024)
		for i := 0; i < 10; i++ {
			xc.scc.CGO(xc.worker)
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
	if !xc.scc.Cancel() {
		return
	}
	for _, c := range xc.clients {
		if err = c.client.Close(); err != nil {
			return
		}
	}
	return xc.scc.Wait(time.Second)
}

func (xc *XClient) Reload() (err error) {
	xc.mutex.Lock()
	defer xc.mutex.Unlock()
	return xc.reload()
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
func (xc *XClient) XCall(ctx context.Context, servicePath, serviceMethod string, args any, reply any) error {
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
func (xc *XClient) Async(ctx context.Context, servicePath, serviceMethod string, args any) (done *client.Call, err error) {
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
		data, err = xc.Binder.Marshal(args)
	}
	if err != nil {
		return nil, err
	}
	return c.Go(ctx, serviceMethod, data, nil, nil)
}

func (xc *XClient) CallWithMetadata(req, res xshare.Metadata, servicePath, serviceMethod string, args, reply any) (err error) {
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

	handler, ok := node.Service.Handler.(*xshare.Handler)
	if !ok {
		logger.Debug("XClient Service handler unknown")
		return
	}

	c := xshare.NewContext(&Context{Message: msg}, xc.Binder)
	if _, err := handler.Caller(node, c); err != nil {
		logger.Debug(err)
	}
}

func (xc *XClient) reload() (err error) {
	for name, value := range xshare.Options.Service {
		if selector := xc.selector(name, value); selector == nil {
			return values.Errorf(0, "Service config error:%v %v", name, value)
		} else if _, err = xc.addServicePath(name, selector); err != nil {
			return
		}
	}
	return
}

func (xc *XClient) selector(k, v string) (r any) {
	if strings.ToLower(v) == xshare.SelectorTypeDiscovery {
		return xshare.NewSelector(k)
	} else if strings.Contains(v, ",") {
		r = strings.Split(v, ",")
	} else {
		r = v
	}
	return
}
