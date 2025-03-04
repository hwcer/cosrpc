package xclient

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosrpc/inprocess"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/smallnest/rpcx/client"
	"sync/atomic"
)

type Client struct {
	client   client.XClient
	Option   client.Option
	started  int32
	FailMode client.FailMode
	Selector interface{} //client.Selector OR client.SelectMode OR address(Peer2Peer MultipleServers)
	//Discovery   client.ServiceDiscovery
	ServicePath string
	//ch          chan *protocol.Message
}

func (this *Client) Start(discovery Discovery) (err error) {
	if !atomic.CompareAndSwapInt32(&this.started, 0, 1) {
		return fmt.Errorf("client started:%v", this.ServicePath)
	}
	switch v := this.Selector.(type) {
	case string:
		if v == xshare.SelectorTypeProcess {
			this.client = inprocess.NewClient(this.ServicePath)
		} else {
			err = this.Peer2Peer(v)
		}
	case []string:
		err = this.Multiple(v)
	case client.Selector:
		err = this.Registry(client.SelectByUser, v, discovery)
	case client.SelectMode:
		err = this.Registry(v, nil, discovery)
	default:
		err = fmt.Errorf("XClient AddServicePath arg(selector) type error:%v", this.Selector)
	}
	return
}
func (this *Client) Close() error {
	return this.client.Close()
}

// Peer2Peer 点对点
func (this *Client) Peer2Peer(address string) error {
	discovery, err := client.NewPeer2PeerDiscovery(xshare.AddressFormat(address), "")
	if err != nil {
		return err
	}
	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, discovery, this.Option)
	return nil
}

// Multiple 点对多
func (this *Client) Multiple(address []string) error {
	var pairs []*client.KVPair
	for _, addr := range address {
		pairs = append(pairs, &client.KVPair{Key: xshare.AddressFormat(addr)})
	}
	discovery, err := client.NewMultipleServersDiscovery(pairs)
	if err != nil {
		return err
	}

	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, discovery, this.Option)
	return nil
}

// Registry 使用注册中心
func (this *Client) Registry(selectMod client.SelectMode, selector client.Selector, discovery Discovery) error {
	if discovery == nil {
		return errors.New("discovery is nil")
	}
	dis, err := discovery(this.ServicePath)
	if err != nil {
		return err
	}

	this.client = client.NewXClient(this.ServicePath, this.FailMode, selectMod, dis, this.Option)
	if selectMod == client.SelectByUser && selector != nil {
		this.client.SetSelector(selector)
	}
	return nil
}
