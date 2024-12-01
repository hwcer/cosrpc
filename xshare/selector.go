package xshare

import (
	"context"
	"fmt"
	"github.com/hwcer/cosgo/options"
	"github.com/smallnest/rpcx/share"
	"net/url"
	"strconv"
	"strings"
)

const (
	ServicesServerIdAll = "-"
)

func NewSelector(servicePath string) *Selector {
	return &Selector{servicePath: servicePath}
}

type node struct {
	Address  string   //tcp@127.0.0.1:8000
	Average  int      //负载
	ServerId []string //服务器
}

type Selector struct {
	services    map[string][]*node //servicePath/serverid   ->service
	servicePath string
}

// Select 默认按负载
func (this *Selector) Select(ctx context.Context, servicePath, serviceMethod string, args interface{}) string {
	metadata, _ := ctx.Value(share.ReqMetaDataKey).(map[string]string)
	serverId := ServicesServerIdAll
	if metadata != nil {
		if address, ok := metadata[options.ServicesSelectorServerAddress]; ok {
			return AddressFormat(address)
		}
		if v, ok := metadata[options.ServicesSelectorServerId]; ok {
			serverId = v
		}
	}

	list := this.services[serverId]
	if len(list) == 0 {
		return ""
	} else if len(list) == 1 {
		return list[0].Address
	}

	var s *node
	for _, v := range list {
		if s == nil || v.Average < s.Average {
			s = v
		}
	}
	s.Average += 1
	return s.Address
}

func (this *Selector) UpdateServer(servers map[string]string) {
	ss := make(map[string][]*node)
	//logger.Debug("===================UpdateServer:%v============================", this.servicePath)
	prefix := fmt.Sprintf("%v/%v/", options.Options.Appid, this.servicePath)
	for address, value := range servers {
		if !strings.HasPrefix(address, prefix) {
			continue
		}
		//logger.Debug("UpdateServer  address：%v value:%v", address, value)
		s := &node{}
		s.Address = strings.TrimPrefix(address, prefix)
		if query, err := url.ParseQuery(value); err == nil {
			s.Average, _ = strconv.Atoi(query.Get(options.ServicesSelectorAverage))
			s.ServerId = strings.Split(query.Get(options.ServicesSelectorServerId), ",")
		}
		for _, k := range s.ServerId {
			ss[k] = append(ss[k], s)
		}
		ss[ServicesServerIdAll] = append(ss[ServicesServerIdAll], s)
	}
	this.services = ss
}
