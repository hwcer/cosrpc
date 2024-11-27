package xshare

import (
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/options"
	"github.com/hwcer/cosgo/utils"
	"strings"
	"time"
)

var Binder binder.Interface = binder.Json

var rpcServerAddress *utils.Address

func Address() *utils.Address {
	if rpcServerAddress != nil {
		return rpcServerAddress
	}
	rpcServerAddress = utils.NewAddress(options.Rpcx.Address)
	if rpcServerAddress.Retry == 0 {
		rpcServerAddress.Retry = 100
	}
	if rpcServerAddress.Host == "" {
		rpcServerAddress.Host, _ = LocalIpv4()
	}
	rpcServerAddress.Scheme = options.Rpcx.Network
	return rpcServerAddress
}

func Timeout() time.Duration {
	return time.Second * time.Duration(options.Rpcx.Timeout)
}

func AddressPrefix() string {
	return options.Rpcx.Network + "@"
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
	for _, s := range ipv4 {
		i := strings.Index(s, ".")
		if k := s[:i]; k == "192" || k == "10" || k == "172" {
			return s, nil
		}
	}
	if len(ipv4) == 0 {
		err = fmt.Errorf("无法获取服务器的内网IP")
	} else {
		ip = ipv4[0]
	}
	return
}
