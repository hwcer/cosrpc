package client

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosrpc"
	"github.com/hwcer/cosrpc/inprocess"
	"github.com/smallnest/rpcx/client"
	"sync/atomic"
)

type Client struct {
	client      client.XClient
	Option      client.Option
	started     int32
	FailMode    client.FailMode
	Selector    interface{} //client.Selector OR client.SelectMode OR address(Peer2Peer MultipleServers)
	ServicePath string
}

func (this *Client) start() (err error) {
	if !atomic.CompareAndSwapInt32(&this.started, 0, 1) {
		return fmt.Errorf("client started:%v", this.ServicePath)
	}
	switch v := this.Selector.(type) {
	case string:
		if v == cosrpc.SelectorTypeProcess {
			this.client = inprocess.NewClient(this.ServicePath)
		} else {
			err = this.Peer2Peer(v)
		}
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

func (this *Client) close() error {
	return this.client.Close()
}

// Peer2Peer 点对点
func (this *Client) Peer2Peer(address string) error {
	dis, err := client.NewPeer2PeerDiscovery(cosrpc.AddressFormat(address), "")
	if err != nil {
		return err
	}
	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, dis, this.Option)
	return nil
}

// Multiple 点对多
func (this *Client) Multiple(address []string) error {
	var pairs []*client.KVPair
	for _, addr := range address {
		pairs = append(pairs, &client.KVPair{Key: cosrpc.AddressFormat(addr)})
	}
	dis, err := client.NewMultipleServersDiscovery(pairs)
	if err != nil {
		return err
	}

	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, dis, this.Option)
	return nil
}

// Registry 使用注册中心
func (this *Client) Registry(selectMod client.SelectMode, selector client.Selector) error {
	if discoveryDefault == nil {
		return errors.New("discovery is nil")
	}
	dis, err := discoveryDefault(this.ServicePath)
	if err != nil {
		return err
	}

	this.client = client.NewXClient(this.ServicePath, this.FailMode, selectMod, dis, this.Option)
	if selectMod == client.SelectByUser && selector != nil {
		this.client.SetSelector(selector)
	}
	return nil
}
