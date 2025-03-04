package xshare

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/utils"
	"strings"
	"time"
)

var Binder binder.Binder = binder.Json
var rpcServerAddress *utils.Address

var Options = &Rpcx{
	Timeout:             10,
	Network:             "tcp",
	Address:             ":8100",
	BasePath:            "cosrpc",
	ClientMessageChan:   300,
	ClientMessageWorker: 1,
}

type Rpcx = struct {
	//Redis               string //服务发现
	Timeout             int32
	Network             string
	Address             string //仅仅启动服务器时需要
	BasePath            string
	ClientMessageChan   int //双向通信客户端接受消息通道大小
	ClientMessageWorker int //双向通信客户端处理消息协程数量
}

func Address() *utils.Address {
	if rpcServerAddress != nil {
		return rpcServerAddress
	}
	rpcServerAddress = utils.NewAddress(Options.Address)
	if rpcServerAddress.Retry == 0 {
		rpcServerAddress.Retry = 100
	}
	if rpcServerAddress.Host == "" {
		rpcServerAddress.Host = "0.0.0.0"
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
