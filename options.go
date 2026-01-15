package cosrpc

import (
	"strings"
	"time"

	"github.com/hwcer/cosgo/utils"
)

var rpcServerAddress *utils.Address

var Config = &Options{
	Timeout:             10,
	Network:             "tcp",
	Address:             ":8100",
	ClientMessageChan:   300,
	ClientMessageWorker: 1,
}

type Options = struct {
	Timeout             int32  `json:"timeout"`
	Network             string `json:"network"`
	Address             string `json:"address"` //仅仅启动服务器时需要
	ClientMessageChan   int    //双向通信客户端接受消息通道大小
	ClientMessageWorker int    //双向通信客户端处理消息协程数量
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
