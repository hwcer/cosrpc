package cosrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/message"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"reflect"
	"sync/atomic"
)

func NewXClient(discovery client.ServiceDiscovery) *XClient {
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
	Discovery client.ServiceDiscovery
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
	c.Option.SerializeType = protocol.SerializeNone
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
		var discovery client.ServiceDiscovery
		if c.Discovery != nil {
			discovery = c.Discovery
		} else {
			discovery = this.Discovery
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

		c.client = client.NewXClient(servicePath, c.FailMode, selectMod, discovery, c.Option)
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
		data, err = json.Marshal(args)
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
	err = json.Unmarshal(v, msg)
	if err != nil {
		return err
	}
	if msg.Code != 0 {
		return msg
	}
	return msg.Unmarshal(reply)
}
