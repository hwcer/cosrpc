package cosrpc

import (
	"fmt"
	"github.com/smallnest/rpcx/client"
)

type Client struct {
	client      client.XClient
	Option      client.Option
	FailMode    client.FailMode
	Selector    interface{} //client.Selector OR client.SelectMode OR address(Peer2Peer MultipleServers)
	Discovery   client.ServiceDiscovery
	ServicePath string
}

func (this *Client) Start(discovery client.ServiceDiscovery) (err error) {
	if this.Discovery == nil {
		this.Discovery = discovery
	}
	switch v := this.Selector.(type) {
	case string:
		err = this.Peer2Peer(v)
	case []string:
		err = this.Multiple(v)
	case client.Selector:
		err = this.Registry(client.SelectByUser, v)
	case client.SelectMode:
		err = this.Registry(v, nil)
	default:
		err = fmt.Errorf("XClient AddServicePath arg(selector) type error:%v", this.Selector)
	}
	return
}

// Peer2Peer 点对点
func (this *Client) Peer2Peer(address string) error {
	discovery, err := client.NewPeer2PeerDiscovery("tcp@"+address, "")
	if err != nil {
		return err
	}
	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, discovery, this.Option)
	return nil
}

// Multiple 点对多
func (this *Client) Multiple(address []string) error {
	var arr []*client.KVPair
	for _, addr := range address {
		arr = append(arr, &client.KVPair{Key: fmt.Sprintf("tcp@%v", addr)})
	}
	discovery, err := client.NewMultipleServersDiscovery(arr)
	if err != nil {
		return err
	}
	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, discovery, this.Option)
	return nil
}

// Registry 使用注册中心
func (this *Client) Registry(selectMod client.SelectMode, selector client.Selector) error {
	this.client = client.NewXClient(this.ServicePath, this.FailMode, selectMod, this.Discovery, this.Option)
	if selectMod == client.SelectByUser && selector != nil {
		this.client.SetSelector(selector)
	}
	return nil
}
