package xshare

import (
	"fmt"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosrpc/redis"
	"github.com/rpcxio/libkv/store"
	"github.com/smallnest/rpcx/client"
	"net/url"
	"strings"
	"time"
)

var rpcxRegister *redis.Register
var rpcxDiscovery client.ServiceDiscovery

func Discovery() (client.ServiceDiscovery, error) {
	servicePath := "/"
	if rpcxDiscovery != nil {
		return rpcxDiscovery, nil
	}
	address, opt, err := getRedisAddress()
	if err != nil {
		return nil, err
	}
	rpcxDiscovery, err = redis.NewDiscovery(Options.BasePath, servicePath, address, opt)
	if err != nil {
		return nil, err
	}
	rpcxDiscovery.SetFilter(serviceDiscoveryFilter)
	return rpcxDiscovery, nil
}

func Register(urlRpcxAddr *utils.Address) (*redis.Register, error) {
	if rpcxRegister != nil {
		return rpcxRegister, nil
	}
	address, opt, err := getRedisAddress()
	if err != nil {
		return nil, err
	}
	host := urlRpcxAddr.Host
	if utils.LocalValid(host) {
		host, err = utils.LocalIPv4()
	}
	if err != nil {
		return nil, err
	}
	rpcxRegister = &redis.Register{
		ServiceAddress: fmt.Sprintf("%v%v:%v", AddressPrefix(), host, urlRpcxAddr.Port),
		RedisServers:   address,
		BasePath:       Options.BasePath,
		Options:        opt,
		UpdateInterval: time.Second * 10,
	}
	return rpcxRegister, nil
}

func serviceDiscoveryFilter(kv *client.KVPair) bool {
	return strings.Contains(kv.Key, AddressPrefix())
}

func getRedisAddress() (address []string, opts *store.Config, err error) {
	var uri *url.URL
	uri, err = utils.NewUrl(Options.Redis, "tcp")
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
