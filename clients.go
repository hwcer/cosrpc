package cosrpc

import (
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosrpc/xclient"
	"strings"
)

var Client *clients

func init() {
	Client = &clients{watches: map[string]bool{}}
	Client.XClient = xclient.NewXClient()
}

type clients struct {
	*xclient.XClient
	watches map[string]bool
}

func (this *clients) Start() (err error) {
	this.XClient.Discovery = getRpcDiscovery
	if err = this.Reload(); err != nil {
		return
	}
	return this.XClient.Start()
}

func (this *clients) Close() error {
	return this.XClient.Close()
}
func (this *clients) Has(name string) bool {
	return this.watches[name]
}

// Watch 关注的服务，必须在init函数中调用
func (this *clients) Watch(servicePath ...string) {
	for _, s := range servicePath {
		this.watches[s] = true
	}
}

func (this *clients) Reload() (err error) {
	for name, _ := range this.watches {
		if selector := this.selector(name, Options.Service[name]); selector == nil {
			return values.Errorf(0, "service not exist:%v", name)
		} else if _, err = this.XClient.AddServicePath(name, selector); err != nil {
			return
		}
	}
	return
}

func (this *clients) selector(k, v string) (r any) {
	if strings.ToLower(v) == SelectorTypeLocal {
		r = Server.Address().String()
	} else if strings.ToLower(v) == SelectorTypeDiscovery {
		return NewSelector(k)
	} else if strings.Contains(v, ",") {
		r = strings.Split(v, ",")
	} else {
		r = v
	}
	return
}
