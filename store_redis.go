package cosrpc

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

var rpcxRegister *redis.RedisRegisterPlugin
var rpcxDiscovery client.ServiceDiscovery

func getRpcRegister(urlRpcxAddr *utils.Address, basePath string) (*redis.RedisRegisterPlugin, error) {
	if rpcxRegister != nil {
		return rpcxRegister, nil
	}
	address, options, err := getRedisAddress()
	if err != nil {
		return nil, err
	}
	rpcxRegister = &redis.RedisRegisterPlugin{
		ServiceAddress: fmt.Sprintf("%v%v:%v", RpcAddressPrefix(), urlRpcxAddr.Host, urlRpcxAddr.Port),
		RedisServers:   address,
		BasePath:       basePath,
		Options:        options,
		UpdateInterval: time.Second * 10,
	}
	return rpcxRegister, nil
}

func getRpcDiscovery() (client.ServiceDiscovery, error) {
	servicePath := "/"
	if rpcxDiscovery != nil {
		return rpcxDiscovery, nil
	}
	address, options, err := getRedisAddress()
	if err != nil {
		return nil, err
	}
	rpcxDiscovery, err = redis.NewRedisDiscovery(Options.Rpcx.BasePath, servicePath, address, options)
	if err != nil {
		return nil, err
	}
	rpcxDiscovery.SetFilter(serviceDiscoveryFilter)
	return rpcxDiscovery, nil
}

func serviceDiscoveryFilter(kv *client.KVPair) bool {
	return strings.Contains(kv.Key, RpcAddressPrefix())
}

func getRedisAddress() (address []string, opts *store.Config, err error) {
	var uri *url.URL
	uri, err = utils.NewUrl(Options.Rpcx.Redis, "tcp")
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
