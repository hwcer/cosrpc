package inprocess

import (
	"bytes"
	"context"
	"errors"
	"github.com/hwcer/cosrpc/xserver"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"
	"io"
	"net"
	"reflect"
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
func (c *Client) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) (err error) {
	if reply != nil && reflect.TypeOf(reply).Kind() != reflect.Ptr {
		return errors.New("client.call reply must pointer")
	}
	node, ok := xserver.Default.Registry.Match(c.servicePath, serviceMethod)
	if !ok {
		return errors.New("services not found: " + serviceMethod)
	}

	req := &Request{}
	req.ServicePath = c.servicePath
	req.ServiceMethod = serviceMethod
	if req.Payload, err = xshare.Marshal(args); err != nil {
		return err
	}
	sc := &Context{req: req, meta: map[any]any{}}
	sc.reply = bytes.Buffer{}
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

	if err = xserver.Default.Caller(sc, node); err != nil {
		return err
	}
	if reply == nil {
		return nil
	}
	switch r := reply.(type) {
	case *[]byte:
		*r = sc.reply.Bytes()
	case *string:
		*r = sc.reply.String()
	default:
		err = xshare.Binder.Unmarshal(sc.reply.Bytes(), reply)
	}
	return err
}

func (c *Client) Oneshot(ctx context.Context, serviceMethod string, args interface{}) error {
	return nil
}
func (c *Client) Broadcast(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	return nil
}
func (c *Client) Fork(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
	return nil
}
func (c *Client) Inform(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) ([]client.Receipt, error) {
	return nil, nil
}
func (c *Client) SendRaw(ctx context.Context, r *protocol.Message) (map[string]string, []byte, error) {
	return nil, nil, nil
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
