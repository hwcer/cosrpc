package redis

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosrpc"
	xclient "github.com/hwcer/cosrpc/client"
	"github.com/hwcer/cosrpc/selector"
	xserver "github.com/hwcer/cosrpc/server"
	"github.com/rpcxio/libkv/store"
	"github.com/smallnest/rpcx/client"
)

type Rpcx struct {
	*cosrpc.Options `json:",inline" mapstructure:",squash"`
	Redis           string `json:"redis" mapstructure:"redis"`
}

var Options = struct {
	Rpcx    *Rpcx             `json:"rpcx"`
	Appid   string            `json:"appid" mapstructure:"appid"`
	Service map[string]string `json:"service"`
}{
	Rpcx:    &Rpcx{Options: cosrpc.Config},
	Appid:   "cosrpc",
	Service: cosrpc.Service,
}

// Start 使用 redis 作为服务器发现 启动RPC功能
func Start() (err error) {
	if err = cosgo.Config.Unmarshal(&Options); err != nil {
		return
	}
	for k, v := range Options.Service {
		if v == cosrpc.SelectorTypeDiscovery {
			cosrpc.Selector.Set(k, selector.New(k))
		}
	}

	if Options.Rpcx.Redis != "" {
		xserver.SetRegister(GetRegister)
		xclient.SetDiscovery(GetDiscovery)
	}

	return nil
}

func GetDiscovery(servicePath string) (client.ServiceDiscovery, error) {
	address, opt, err := rpcxRedisParse()
	if err != nil {
		return nil, err
	}
	var discovery *Discovery
	discovery, err = NewDiscovery(Options.Appid, servicePath, address, opt)
	if err != nil {
		return nil, err
	}
	return discovery, nil
}

func GetRegister() (xserver.Register, error) {
	rpcxAddr := cosrpc.Address()
	address, opt, err := rpcxRedisParse()
	if err != nil {
		return nil, err
	}
	host := rpcxAddr.Host
	if utils.LocalValid(host) {
		host, err = utils.LocalIPv4()
	}
	if err != nil {
		return nil, err
	}
	rpcxRegister := &Register{
		ServiceAddress: fmt.Sprintf("%v%v:%v", cosrpc.AddressPrefix(), host, rpcxAddr.Port),
		RedisServers:   address,
		BasePath:       Options.Appid,
		Options:        opt,
		UpdateInterval: time.Second,
	}
	return rpcxRegister, nil
}

func rpcxRedisAddress() (addr string, err error) {
	if Options.Rpcx.Redis == "" {
		return "", fmt.Errorf("rpcx redis address is empty")
	}
	return Options.Rpcx.Redis, nil
}

func rpcxRedisParse() (address []string, opts *store.Config, err error) {
	var addr string
	if addr, err = rpcxRedisAddress(); err != nil {
		return
	} else if addr == "" {
		return nil, nil, errors.New("redis address is empty")
	}
	var uri *url.URL
	uri, err = utils.NewUrl(addr, "tcp")
	if err != nil {
		return
	}
	address = []string{uri.Host}
	opts = &store.Config{}
	query := uri.Query()
	opts.Password = query.Get("password")
	if query.Has("db") {
		opts.Bucket = query.Get("db")
	} else {
		opts.Bucket = "13"
	}
	return
}
