package cosrpc

import (
	"github.com/hwcer/cosgo/utils"
	"strings"
	"time"
)

var rpcServerAddress *utils.Address

var Config = &Options{
	Timeout:             10,
	Network:             "tcp",
	Address:             ":8100",
	BasePath:            "cosrpc",
	ClientMessageChan:   300,
	ClientMessageWorker: 1,
}

type Options = struct {
	Timeout             int32
	Network             string
	Address             string //仅仅启动服务器时需要
	BasePath            string
	ClientMessageChan   int //双向通信客户端接受消息通道大小
	ClientMessageWorker int //双向通信客户端处理消息协程数量
}

func SetBasePath(p string) {
	Config.BasePath = p
}

func Address() *utils.Address {
	if rpcServerAddress != nil {
		return rpcServerAddress
	}
	rpcServerAddress = utils.NewAddress(Config.Address)
	if rpcServerAddress.Retry == 0 {
		rpcServerAddress.Retry = 100
	}
	if rpcServerAddress.Host == "" {
		rpcServerAddress.Host = "0.0.0.0"
	}
	rpcServerAddress.Scheme = Config.Network
	return rpcServerAddress
}

func Timeout() time.Duration {
	return time.Second * time.Duration(Config.Timeout)
}

func AddressPrefix() string {
	return Config.Network + "@"
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
