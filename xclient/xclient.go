package xclient

import (
	"context"
	"fmt"
	_ "github.com/hwcer/cosrpc/logger"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/client"
	"sync/atomic"
)

type Discovery func(servicePath string) (client.ServiceDiscovery, error)

func NewXClient(discovery Discovery) *XClient {
	return &XClient{
		clients:   make(map[string]*Client),
		Discovery: discovery,
	}
}

type Client struct {
	client    client.XClient
	Option    client.Option
	FailMode  client.FailMode
	Selector  interface{} //client.Selector OR client.SelectMode
	Discovery client.ServiceDiscovery
}

type XClient struct {
	start     int32
	clients   map[string]*Client
	Discovery Discovery
}

// AddServicePath 观察服务器信息
func (this *XClient) AddServicePath(servicePath string, selector interface{}) *Client {
	if atomic.LoadInt32(&this.start) > 0 {
		logger.Fatal("Client已经启动无法再添加Service")
	}
	c := &Client{}
	this.clients[servicePath] = c
	c.Option = client.DefaultOption
	c.FailMode = client.Failtry
	c.Selector = selector
	return c
}

func (this *XClient) Start() (err error) {
	if !atomic.CompareAndSwapInt32(&this.start, 0, 1) {
		return
	}
	if len(this.clients) == 0 {
		return
	}
	for servicePath, c := range this.clients {
		if c.Discovery == nil {
			if c.Discovery, err = this.Discovery(servicePath); err != nil {
				return
			}
		}
		var selector client.Selector
		var selectMod client.SelectMode
		switch c.Selector.(type) {
		case client.Selector:
			selector = c.Selector.(client.Selector)
			selectMod = client.SelectByUser
		case client.SelectMode:
			selectMod = c.Selector.(client.SelectMode)
		default:
			logger.Fatal("XClient AddServicePath arg(Selector) type error:%v", selector)
		}

		c.client = client.NewXClient(servicePath, c.FailMode, selectMod, c.Discovery, c.Option)
		if selectMod == client.SelectByUser && selector != nil {
			c.client.SetSelector(selector)
		}
	}
	return
}

func (this *XClient) Close() (errs []error) {
	var err error
	for _, c := range this.clients {
		if err = c.client.Close(); err != nil {
			errs = append(errs, err)
		}
		c.Discovery.Close()
	}
	return nil
}

func (this *XClient) Has(servicePath string) bool {
	if _, ok := this.clients[servicePath]; ok {
		return true
	} else {
		return false
	}
}

func (this *XClient) Size() int {
	return len(this.clients)
}

func (this *XClient) Client(servicePath string) client.XClient {
	if v, ok := this.clients[servicePath]; ok {
		return v.client
	} else {
		return nil
	}
}

func (this *XClient) Call(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c := this.Client(servicePath); c != nil {
		return c.Call(ctx, serviceMethod, args, reply)
	} else {
		return fmt.Errorf("服务不存在")
	}
}

func (this *XClient) Broadcast(ctx context.Context, servicePath, serviceMethod string, args, reply interface{}) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c := this.Client(servicePath); c != nil {
		return c.Broadcast(ctx, serviceMethod, args, reply)
	} else {
		return fmt.Errorf("服务不存在")
	}
}
