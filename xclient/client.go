package xclient

import (
	"fmt"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"sync/atomic"
)

type Client struct {
	start       int32
	client      client.XClient
	Option      client.Option
	FailMode    client.FailMode
	Selector    interface{} //client.Selector OR client.SelectMode OR address(Peer2Peer MultipleServers)
	Discovery   client.ServiceDiscovery
	ServicePath string
}

func (this *Client) Start(discovery Discovery, ch chan *protocol.Message) (err error) {
	if !atomic.CompareAndSwapInt32(&this.start, 0, 1) {
		return fmt.Errorf("client started:%v", this.ServicePath)
	}
	switch v := this.Selector.(type) {
	case string:
		err = this.Peer2Peer(v, ch)
	case []string:
		err = this.Multiple(v, ch)
	case client.Selector:
		err = this.Registry(client.SelectByUser, v, discovery, ch)
	case client.SelectMode:
		err = this.Registry(v, nil, discovery, ch)
	default:
		err = fmt.Errorf("XClient AddServicePath arg(selector) type error:%v", this.Selector)
	}
	return
}

// Peer2Peer 点对点
func (this *Client) Peer2Peer(address string, ch chan *protocol.Message) error {
	discovery, err := client.NewPeer2PeerDiscovery("tcp@"+address, "")
	if err != nil {
		return err
	}
	if ch != nil {
		this.client = client.NewBidirectionalXClient(this.ServicePath, this.FailMode, client.RandomSelect, discovery, this.Option, ch)
	} else {
		this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, discovery, this.Option)
	}
	return nil
}

// Multiple 点对多
func (this *Client) Multiple(address []string, ch chan *protocol.Message) (err error) {
	var pairs []*client.KVPair
	for _, addr := range address {
		pairs = append(pairs, &client.KVPair{Key: fmt.Sprintf("tcp@%v", addr)})
	}
	if c := this.client; c != nil {
		if discovery, ok := this.Discovery.(*client.MultipleServersDiscovery); ok {
			discovery.Update(pairs)
			return nil
		} else {
			defer func() {
				_ = c.Close()
			}()
		}
	}
	if this.Discovery, err = client.NewMultipleServersDiscovery(pairs); err != nil {
		return
	}

	if ch != nil {
		this.client = client.NewBidirectionalXClient(this.ServicePath, this.FailMode, client.RandomSelect, this.Discovery, this.Option, ch)
	} else {
		this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, this.Discovery, this.Option)
	}

	return
}

// Registry 使用注册中心
func (this *Client) Registry(selectMod client.SelectMode, selector client.Selector, registry Discovery, ch chan *protocol.Message) (err error) {
	if c := this.client; c != nil {
		defer func() {
			_ = c.Close()
		}()
	}
	if this.Discovery, err = registry(); err != nil {
		return
	}
	if ch != nil {
		this.client = client.NewBidirectionalXClient(this.ServicePath, this.FailMode, selectMod, this.Discovery, this.Option, ch)
	} else {
		this.client = client.NewXClient(this.ServicePath, this.FailMode, selectMod, this.Discovery, this.Option)
	}
	if selectMod == client.SelectByUser && selector != nil {
		this.client.SetSelector(selector)
	}
	return
}
