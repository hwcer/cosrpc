package xshare

import (
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/utils"
	"strings"
	"time"
)

const (
	SelectorTypeLocal     = "local"     //本地程序内访问
	SelectorTypeProcess   = "process"   //进程内访问
	SelectorTypeDiscovery = "discovery" //服务发现
)

var Binder = binder.New(binder.MIMEJSON)

type Rpcx struct {
	Redis    string //服务发现
	Timeout  int32
	Network  string
	Address  string //仅仅启动服务器时需要
	BasePath string
}

// Options Etcd Redis 二选一
var Options = &Rpcx{
	Timeout: 2,
	Network: "tcp",
	Address: ":8100",
}

var Service = map[string]string{}

var rpcServerAddress *utils.Address

func Address() *utils.Address {
	if rpcServerAddress != nil {
		return rpcServerAddress
	}
	rpcServerAddress = utils.NewAddress(Options.Address)
	if rpcServerAddress.Retry == 0 {
		rpcServerAddress.Retry = 100
	}
	if rpcServerAddress.Host == "" {
		rpcServerAddress.Host, _ = LocalIpv4()
	}
	rpcServerAddress.Scheme = Options.Network
	return rpcServerAddress
}

func Timeout() time.Duration {
	return time.Second * time.Duration(Options.Timeout)
}

func AddressPrefix() string {
	return Options.Network + "@"
}

func AddressFormat(address string) string {
	prefix := AddressPrefix()
	if strings.HasPrefix(address, prefix) {
		return address
	}
	b := strings.Builder{}
	b.WriteString(prefix)
	b.WriteString(address)
	return b.String()
}

func LocalIpv4() (ip string, err error) {
	var ipv4 []string
	if ipv4, err = utils.LocalIPv4s(); err != nil {
		return
	}
	if len(ipv4) == 0 {
		err = fmt.Errorf("无法获取服务器的内网IP")
	} else {
		ip = ipv4[0]
	}
	return
}
