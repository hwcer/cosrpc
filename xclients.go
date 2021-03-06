package cosrpc

import (
	"context"
	"fmt"
	"github.com/hwcer/cosgo/logger"
	rpcx "github.com/smallnest/rpcx/client"
	"sync/atomic"
)

func NewXClient() *XClient {
	return &XClient{
		clients: make(map[string]*Options),
	}
}

type Options struct {
	client    rpcx.XClient
	options   *rpcx.Option
	selector  rpcx.Selector
	selectMod rpcx.SelectMode
	discovery rpcx.ServiceDiscovery
	FailMode  rpcx.FailMode
}

type XClient struct {
	start     int32
	clients   map[string]*Options
	discovery rpcx.ServiceDiscovery
}

//AddServicePath 观察服务器信息
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
	opts.options = &r
	opts.FailMode = rpcx.Failtry
	return opts.options
}

func (this *XClient) Start(discovery rpcx.ServiceDiscovery) (err error) {
	if !atomic.CompareAndSwapInt32(&this.start, 0, 1) {
		return
	}
	if len(this.clients) == 0 {
		return
	}
	this.discovery = discovery
	for k, _ := range this.clients {
		if err = this.create(k); err != nil {
			return
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
	this.discovery.Close()
	return nil
}

func (this *XClient) create(servicePath string) (err error) {
	c := this.clients[servicePath]
	c.discovery = this.discovery
	//if c.discovery, err = this.discovery.Clone(servicePath); err != nil {
	//	return
	//}
	c.client = rpcx.NewXClient(servicePath, c.FailMode, c.selectMod, c.discovery, *c.options)
	if c.selectMod == rpcx.SelectByUser {
		c.client.SetSelector(c.selector)
	}
	return
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
