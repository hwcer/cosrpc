package cosrpc

import (
	"fmt"
	"github.com/hwcer/cosgo/utils"
	"strings"
)

const (
	SelectorTypeLocal     = "local"     //单机版
	SelectorTypeProcess   = "process"   //进程内访问
	SelectorTypeDiscovery = "discovery" //服务发现
)

// Options Etcd Redis 二选一
var Options = &struct {
	Rpcx    *rpcx
	Service map[string]string
}{
	Rpcx:    &rpcx{Network: "tcp", Address: ":8100"},
	Service: make(map[string]string),
}

type rpcx struct {
	Redis    string //服务发现
	Network  string
	Address  string
	BasePath string
}

var serverAddressPrefix string

func RpcAddressPrefix() string {
	if serverAddressPrefix == "" {
		serverAddressPrefix = Options.Rpcx.Network + "@"
	}
	return serverAddressPrefix
}

func RpcAddressFormat(address string) string {
	prefix := RpcAddressPrefix()
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
