package xclient

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosrpc/inprocess"
	"github.com/hwcer/cosrpc/xshare"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/log"
	"github.com/smallnest/rpcx/protocol"
	"sync/atomic"
)

type Client struct {
	client      client.XClient
	Option      client.Option
	started     int32
	FailMode    client.FailMode
	Selector    interface{} //client.Selector OR client.SelectMode OR address(Peer2Peer MultipleServers)
	Discovery   client.ServiceDiscovery
	ServicePath string
	ch          chan *protocol.Message
}

func (this *Client) Start(discovery Discovery, ch chan *protocol.Message) (err error) {
	if !atomic.CompareAndSwapInt32(&this.started, 0, 1) {
		return fmt.Errorf("client started:%v", this.ServicePath)
	}
	this.ch = ch
	switch v := this.Selector.(type) {
	case string:
		if v == xshare.SelectorTypeProcess {
			this.client = inprocess.NewClient(this.ServicePath)
		} else {
			err = this.Multiple([]string{v})
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
func (this *Client) Reload(selector any) (err error) {
	if this.started == 0 {
		this.Selector = selector
		return nil
	}
	switch v := this.Selector.(type) {
	case string:
		err = this.Multiple([]string{v})
	case []string:
		err = this.Multiple(v)
	default:
		err = fmt.Errorf("XClient Reload arg(selector) type error:%v", selector)
	}
	if err != nil {
		log.Infof("XClient Reload error:%v", err)
		err = nil
	}

	return
}

// Peer2Peer 点对点
func (this *Client) Peer2Peer(address string) (err error) {
	if this.Discovery, err = client.NewPeer2PeerDiscovery(xshare.AddressFormat(address), ""); err != nil {
		return err
	}

	if this.ch != nil {
		this.client = client.NewBidirectionalXClient(this.ServicePath, this.FailMode, client.RandomSelect, this.Discovery, this.Option, this.ch)
	} else {
		this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, this.Discovery, this.Option)
	}
	return nil
}

// Multiple 点对多
func (this *Client) Multiple(address []string) (err error) {
	var pairs []*client.KVPair
	for _, addr := range address {
		pairs = append(pairs, &client.KVPair{Key: xshare.AddressFormat(addr)})
	}
	if c := this.client; c != nil {
		if discovery, ok := this.Discovery.(*client.MultipleServersDiscovery); ok {
			discovery.Update(pairs)
		} else {
			err = errors.New("XClient MultiServersDiscovery arg(selector) type error")
		}
		return
	}
	if this.Discovery, err = client.NewMultipleServersDiscovery(pairs); err != nil {
		return
	}

	if this.ch != nil {
		this.client = client.NewBidirectionalXClient(this.ServicePath, this.FailMode, client.RandomSelect, this.Discovery, this.Option, this.ch)
	} else {
		this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, this.Discovery, this.Option)
	}

	return
}

// Registry 使用注册中心
func (this *Client) Registry(selectMod client.SelectMode, selector client.Selector, registry Discovery) (err error) {
	if c := this.client; c != nil {
		defer func() {
			_ = c.Close()
		}()
	}
	if registry != nil {
		if this.Discovery, err = registry(); err != nil {
			return
		}
	}
	if this.Discovery == nil {
		return errors.New("XClient Register arg(registry) empty")
	}

	if this.ch != nil {
		this.client = client.NewBidirectionalXClient(this.ServicePath, this.FailMode, selectMod, this.Discovery, this.Option, this.ch)
	} else {
		this.client = client.NewXClient(this.ServicePath, this.FailMode, selectMod, this.Discovery, this.Option)
	}
	if selectMod == client.SelectByUser && selector != nil {
		this.client.SetSelector(selector)
	}
	return
}
