package selector

import (
	"context"
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/hwcer/cosrpc"
	"github.com/smallnest/rpcx/share"
)

const (
	MetaDataAverage  = "_rpc_srv_avg"
	MetaDataAddress  = "_rpc_srv_addr" //rpc服务器ID,selector 中固定转发地址
	MetaDataServerId = "_rpc_srv_sid"  //服务器编号
)

func New(servicePath string) *Selector {
	s := &Selector{servicePath: servicePath}
	s.snapshot.Store(&selectorSnapshot{
		all:      map[string]*selectorNode{},
		services: map[string][]*selectorNode{},
	})
	return s
}

type selectorNode struct {
	sid     string //服务器
	index   uint64
	Address string //tcp@127.0.0.1:8000
	Average atomic.Int32 //负载:Select 并发读改写,必须原子
}

// selectorSnapshot all/services 必须成对一致(sid 索引引用的就是 all 里的节点),
// 打包进单个指针做整体原子发布,避免"新版all+旧版services"的撕裂窗口
type selectorSnapshot struct {
	all      map[string]*selectorNode
	services map[string][]*selectorNode
}

// Selector rpcx 的 Select 插件:Select 在每个 RPC 请求协程上调用,
// UpdateServer 由服务发现 watch 协程触发——读写两侧全程无共享锁,
// 靠 selectorSnapshot 的 COW+原子发布保证竞争安全(读路径一次 atomic Load)。
type Selector struct {
	snapshot    atomic.Pointer[selectorSnapshot]
	servicePath string
}

func (this *Selector) SelectWithServerId(list []*selectorNode) (r string) {
	var s *selectorNode
	for _, v := range list {
		if s == nil || v.Average.Load() < s.Average.Load() {
			s = v
		}
	}
	if s != nil {
		s.Average.Add(1)
		r = s.Address
	}
	return
}

// Select 默认按负载
func (this *Selector) Select(ctx context.Context, servicePath, serviceMethod string, args any) (r string) {
	metadata, _ := ctx.Value(share.ReqMetaDataKey).(map[string]string)
	snap := this.snapshot.Load()
	if metadata != nil {
		if address, ok := metadata[MetaDataAddress]; ok {
			return cosrpc.AddressFormat(address)
		}
		if v, ok := metadata[MetaDataServerId]; ok {
			return this.SelectWithServerId(snap.services[v])
		}
	}

	var s *selectorNode
	for _, v := range snap.all {
		if s == nil || v.Average.Load() < s.Average.Load() {
			s = v
		}
	}
	if s != nil {
		s.Average.Add(1)
		r = s.Address
	}
	return
}

func (this *Selector) UpdateServer(servers map[string]string) {
	old := this.snapshot.Load()
	all := make(map[string]*selectorNode, len(servers))
	service := make(map[string][]*selectorNode)

	for address, value := range servers {
		s := &selectorNode{}
		s.Address = address
		if v, ok := old.all[address]; ok {
			s.index = v.index
		}
		if query, err := url.ParseQuery(value); err == nil {
			s.sid = query.Get(MetaDataServerId)
			if avg, err := strconv.Atoi(query.Get(MetaDataAverage)); err == nil {
				s.Average.Store(int32(avg))
			}
		}
		all[address] = s
	}
	for _, v := range all {
		if v.sid != "" {
			service[v.sid] = append(service[v.sid], v)
		}
	}
	this.snapshot.Store(&selectorSnapshot{all: all, services: service})
}
