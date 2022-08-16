package xclient

import (
	"context"
	"fmt"
	"github.com/hwcer/cosgo/logger"
	rpcx "github.com/smallnest/rpcx/client"
	"sync/atomic"
)

type Discovery func(servicePath string) (rpcx.ServiceDiscovery, error)

func NewXClient(discovery Discovery) *XClient {
	return &XClient{
		clients:   make(map[string]*Options),
		Discovery: discovery,
	}
}

type Options struct {
	client    rpcx.XClient
	option    *rpcx.Option
	selector  rpcx.Selector
	selectMod rpcx.SelectMode
	discovery rpcx.ServiceDiscovery
	FailMode  rpcx.FailMode
}

type XClient struct {
	start     int32
	clients   map[string]*Options
	Discovery Discovery
}

// AddServicePath 观察服务器信息
func (this *XClient) AddServicePath(servicePath string, selector interface{}) *rpcx.Option {
	if atomic.LoadInt32(&this.start) > 0 {
		logger.Fatal("Client已经启动无法再添加Service")
	}
	opts := &Options{}
	this.clients[servicePath] = opts
	switch selector.(type) {
	case rpcx.Selector:
		opts.selector = selector.(rpcx.Selector)
		opts.selectMod = rpcx.SelectByUser
	case rpcx.SelectMode:
		opts.selectMod = selector.(rpcx.SelectMode)
	default:
		logger.Fatal("XClient AddServicePath arg(Selector) type error:%v", selector)
	}
	r := rpcx.DefaultOption
	opts.option = &r
	opts.FailMode = rpcx.Failtry
	return opts.option
}

func (this *XClient) Start() (err error) {
	if !atomic.CompareAndSwapInt32(&this.start, 0, 1) {
		return
	}
	if len(this.clients) == 0 {
		return
	}
	for servicePath, client := range this.clients {
		if client.discovery, err = this.Discovery(servicePath); err != nil {
			return
		}
		client.client = rpcx.NewXClient(servicePath, client.FailMode, client.selectMod, client.discovery, *client.option)
		if client.selectMod == rpcx.SelectByUser {
			client.client.SetSelector(client.selector)
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
		c.discovery.Close()
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

func (this *XClient) Client(servicePath string) rpcx.XClient {
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
