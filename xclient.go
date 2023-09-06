package cosrpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"reflect"
	"sync"
	"time"
)

// 注册中心服务发现,点对点或者点对多时无需设置
type RegistryDiscovery func() (client.ServiceDiscovery, error)

func NewXClient() *XClient {
	return &XClient{
		clients: make(map[string]*Client),
		Binder:  binder.New(binder.MIMEJSON),
	}
}

type XClient struct {
	mutex     sync.Mutex
	started   bool
	clients   map[string]*Client
	Binder    binder.Interface
	Discovery RegistryDiscovery
}

// AddServicePath 观察服务器信息
func (this *XClient) AddServicePath(servicePath string, selector any) (c *Client, err error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	exist := this.clients[servicePath]
	if exist != nil && Equal(exist.Selector, selector) {
		c = exist
		return
	}
	c = &Client{}
	c.Option = client.DefaultOption
	c.FailMode = client.Failover
	c.Selector = selector
	c.ServicePath = servicePath
	c.Option.SerializeType = protocol.SerializeNone
	if this.started {
		if err = c.Start(this.Discovery); err != nil {
			return
		}
	}
	clients := map[string]*Client{}
	for k, v := range this.clients {
		clients[k] = v
	}
	clients[servicePath] = c
	this.clients = clients

	if exist != nil {
		//服务重定向，关闭旧的client
		time.AfterFunc(5*time.Second, func() {
			_ = exist.client.Close()
		})
	}
	return
}

func (this *XClient) Start() (err error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.started {
		return
	}
	this.started = true
	for _, c := range this.clients {
		if err = c.Start(this.Discovery); err != nil {
			return
		}
	}
	return
}

func (this *XClient) Close() (err error) {
	for _, c := range this.clients {
		if err = c.client.Close(); err != nil {
			return
		}
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
		return fmt.Errorf("can not found any server:%v", servicePath)
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
	if _, ok := reply.(*[]byte); ok {
		if err = this.Call(ctx, servicePath, serviceMethod, data, reply); err != nil {
			//return ParseError(err)
			return err
		} else {
			return nil
		}
	}
	v := make([]byte, 0)
	err = this.Call(ctx, servicePath, serviceMethod, data, &v)

	var msg *values.Message
	var isReplyMsg bool
	if reply == nil {
		msg = values.NewMessage(nil)
	} else if msg, isReplyMsg = reply.(*values.Message); !isReplyMsg {
		msg = values.NewMessage(nil)
	}
	if err == nil {
		err = this.Binder.Unmarshal(v, msg)
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
func (this *XClient) Async(ctx context.Context, servicePath, serviceMethod string, args interface{}) (err error) {
	var data []byte
	if v, ok := args.([]byte); ok {
		data = v
	} else {
		data, err = this.Binder.Marshal(args)
	}
	if err != nil {
		return err
	}
	if c := this.Client(servicePath); c != nil {
		_, err = c.Go(ctx, serviceMethod, data, nil, nil)
	} else {
		err = client.ErrXClientNoServer
	}
	return err
}
