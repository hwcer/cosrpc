package client

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/hwcer/cosrpc"
	"github.com/hwcer/cosrpc/inprocess"
	"github.com/smallnest/rpcx/client"
)

// Client 是 cosrpc 客户端的核心结构
// 封装了 rpcx XClient 并提供了多种服务发现模式
type Client struct {
	client   client.XClient  // 内嵌的 rpcx XClient
	Option   client.Option   // 客户端选项
	started  int32           // 客户端启动状态，0 未启动，1 已启动
	FailMode client.FailMode // 失败处理模式
	Selector interface{}     // 服务选择器，可以是以下类型：
	// - string: 进程内调用或点对点地址
	// - []string: 多点地址列表
	// - client.Selector: 自定义选择器
	// - client.SelectMode: 选择模式
	ServicePath string // 服务路径
}

// start 启动客户端
// 1. 原子操作检查并设置启动状态
// 2. 根据 Selector 类型选择不同的服务发现模式
// 3. 初始化对应的 XClient
func (this *Client) start() (err error) {
	if !atomic.CompareAndSwapInt32(&this.started, 0, 1) {
		return fmt.Errorf("client started:%v", this.ServicePath)
	}
	switch v := this.Selector.(type) {
	case string:
		if v == cosrpc.SelectorTypeProcess {
			// 进程内调用
			this.client = inprocess.NewClient(this.ServicePath)
		} else {
			// 点对点调用
			err = this.Peer2Peer(v)
		}
	case []string:
		// 多点调用
		err = this.Multiple(v)
	case client.Selector:
		// 使用自定义选择器
		err = this.Registry(client.SelectByUser, v)
	case client.SelectMode:
		// 使用选择模式
		err = this.Registry(v, nil)
	default:
		err = fmt.Errorf("XClient AddServicePath arg(selector) type error:%v", this.Selector)
	}
	return
}

// close 关闭客户端
func (this *Client) close() error {
	return this.client.Close()
}

// Peer2Peer 点对点调用模式
// 创建一个点对点的服务发现器并初始化 XClient
func (this *Client) Peer2Peer(address string) error {
	dis, err := client.NewPeer2PeerDiscovery(cosrpc.AddressFormat(address), "")
	if err != nil {
		return err
	}
	this.client = client.NewXClient(this.ServicePath, this.FailMode, client.RandomSelect, dis, this.Option)
	return nil
}

// Multiple 多点调用模式
// 创建一个多点的服务发现器并初始化 XClient
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

// Registry 使用注册中心模式
// 创建一个基于注册中心的服务发现器并初始化 XClient
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
