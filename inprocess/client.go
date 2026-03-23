package inprocess

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"reflect"

	"github.com/hwcer/cosrpc/server"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"
)

type Client struct {
	servicePath string
}

// NewClient creates a XClient that supports service discovery and service governance.
func NewClient(servicePath string) client.XClient {
	c := &Client{
		servicePath: servicePath,
	}
	return c
}
func (c *Client) SetPlugins(plugins client.PluginContainer) {}
func (c *Client) GetPlugins() client.PluginContainer {
	return nil
}
func (c *Client) SetSelector(s client.Selector)                 {}
func (c *Client) ConfigGeoSelector(latitude, longitude float64) {}
func (c *Client) Auth(auth string)                              {}

func (c *Client) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, done chan *client.Call) (*client.Call, error) {
	return nil, nil
}

func (c *Client) Call(ctx context.Context, serviceMethod string, args any, reply any) (err error) {
	if reply != nil && reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return errors.New("client.call reply must pointer")
	}
	node, _ := server.Default.Registry.Search(server.RegistryMethod, c.servicePath, serviceMethod)
	if node == nil {
		return errors.New("services not found: " + serviceMethod)
	}

	req := &Request{}
	req.ServicePath = c.servicePath
	req.ServiceMethod = serviceMethod
	sc := &Context{req: req, meta: map[any]any{}}
	sc.reply = args
	// if req.Payload, err = c.Binder(ctx).Marshal(args); err != nil {
	// 	return err
	// }

	//sc.reply = bytes.Buffer{}
	if v := ctx.Value(share.ReqMetaDataKey); v != nil {
		sc.meta[share.ReqMetaDataKey] = v
	} else {
		sc.meta[share.ReqMetaDataKey] = make(map[string]string)
	}

	if v := ctx.Value(share.ResMetaDataKey); v != nil {
		sc.meta[share.ResMetaDataKey] = v
	} else {
		sc.meta[share.ResMetaDataKey] = make(map[string]string)
	}

	if err = server.Default.Caller(sc, node); err != nil {
		return err
	}
	return Unmarshal(sc.reply, reply)
}

func (c *Client) Oneshot(ctx context.Context, serviceMethod string, args interface{}) error {
	return c.Call(ctx, serviceMethod, args, nil)
}
func (c *Client) Broadcast(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	return c.Call(ctx, serviceMethod, args, reply)
}
func (c *Client) Fork(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	return c.Call(ctx, serviceMethod, args, reply)
}
func (c *Client) Inform(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) ([]client.Receipt, error) {
	err := c.Call(ctx, serviceMethod, args, reply)
	if err != nil {
		return nil, err
	}
	return []client.Receipt{{}}, nil
}
func (c *Client) SendRaw(ctx context.Context, r *protocol.Message) (map[string]string, []byte, error) {
	node, _ := server.Default.Registry.Search(server.RegistryMethod, c.servicePath, r.ServiceMethod)
	if node == nil {
		return nil, nil, errors.New("services not found: " + r.ServiceMethod)
	}

	req := &Request{}
	req.ServicePath = c.servicePath
	req.ServiceMethod = r.ServiceMethod
	sc := &Context{req: req, meta: map[any]any{}}
	sc.reply = r.Payload

	if v := ctx.Value(share.ReqMetaDataKey); v != nil {
		sc.meta[share.ReqMetaDataKey] = v
	} else {
		sc.meta[share.ReqMetaDataKey] = make(map[string]string)
	}

	if v := ctx.Value(share.ResMetaDataKey); v != nil {
		sc.meta[share.ResMetaDataKey] = v
	} else {
		sc.meta[share.ResMetaDataKey] = make(map[string]string)
	}

	if err := server.Default.Caller(sc, node); err != nil {
		return nil, nil, err
	}

	var data []byte
	switch v := sc.reply.(type) {
	case []byte:
		data = v
	case *[]byte:
		data = *v
	default:
		data, _ = json.Marshal(v)
	}

	return sc.Metadata(), data, nil
}
func (c *Client) SendFile(ctx context.Context, fileName string, rateInBytesPerSecond int64, meta map[string]string) error {
	return nil
}

func (c *Client) DownloadFile(ctx context.Context, requestFileName string, saveTo io.Writer, meta map[string]string) error {
	return nil
}

func (c *Client) Stream(ctx context.Context, meta map[string]string) (net.Conn, error) {
	return nil, nil
}

func (c *Client) Close() error {
	return nil
}

// Unmarshal 内存模式专用的反序列化方法
// 类型一致时直接复制，类型不一致时通过 JSON 序列化后反序列化
func Unmarshal(data any, reply any) error {
	if reply == nil {
		return nil
	}

	// 检查类型是否一致
	dataType := reflect.TypeOf(data)
	replyType := reflect.TypeOf(reply).Elem()

	if dataType == replyType {
		// 类型一致，直接赋值
		reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(data))
		return nil
	}

	// 类型不一致，通过 JSON 序列化后反序列化
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, reply)
}
