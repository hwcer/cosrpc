package cosrpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/message"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"reflect"
	"sync"
)

func NewXClient(discovery client.ServiceDiscovery) *XClient {
	return &XClient{
		clients:   make(map[string]*Client),
		Binder:    binder.New(binder.MIMEJSON),
		Discovery: discovery,
	}
}

type XClient struct {
	sync.Mutex
	started   bool
	clients   map[string]*Client
	Binder    binder.Interface
	Discovery client.ServiceDiscovery
}

// AddServicePath 观察服务器信息
func (this *XClient) AddServicePath(servicePath string, selector interface{}) (c *Client, err error) {
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failtry
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	if !this.started {
		this.clients[servicePath] = c
		return
	}
	if err = c.Start(this.Discovery); err != nil {
		return
	}
	exist := this.clients[servicePath]
	clients := make(map[string]*Client)
	for k, v := range this.clients {
		clients[k] = v
	}
	clients[servicePath] = c
	this.clients = clients

	if exist != nil {
		_ = exist.client.Close()
	}

	return
}

func (this *XClient) Start() (err error) {
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	if this.started {
		return nil
	}
	this.started = true
	for _, c := range this.clients {
		if err = c.Start(this.Discovery); err != nil {
			return
		}
	}
	return nil
}

func (this *XClient) Close() (errs []error) {
	var err error
	for _, c := range this.clients {
		if err = c.client.Close(); err != nil {
			errs = append(errs, err)
			if c.Discovery != nil {
				c.Discovery.Close()
			}
		}
	}
	if this.Discovery != nil {
		this.Discovery.Close()
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
		return client.ErrXClientNoServer
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

// XCall 使用默认的message发起请求
func (this *XClient) XCall(ctx context.Context, servicePath, serviceMethod string, args interface{}, reply interface{}) error {
	if reply != nil && reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return errors.New("client.call reply must pointer")
	}
	var err error
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = this.Binder.Marshal(args)
	}
	if err != nil {
		return err
	}
	if _, ok := reply.(*[]byte); ok || reply == nil {
		if err = this.Call(ctx, servicePath, serviceMethod, data, reply); err != nil {
			return ParseError(err)
		} else {
			return nil
		}
	}
	v := make([]byte, 0)
	err = this.Call(ctx, servicePath, serviceMethod, data, &v)
	if err != nil {
		return ParseError(err)
	}

	msg := message.New()
	err = this.Binder.Unmarshal(v, msg)
	if err != nil {
		return err
	}
	if msg.Code != 0 {
		return msg
	}
	return msg.Unmarshal(reply)
}
