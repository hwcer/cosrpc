package redis

import (
	"strings"
	"sync"
	"time"

	"github.com/rpcxio/libkv"
	"github.com/rpcxio/libkv/store"
	"github.com/rpcxio/libkv/store/redis"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/log"
)

var AllowKeyNotFound bool = true

func init() {
	redis.Register()
}

// Discovery is a redis service discovery.
// It always returns the registered servers in redis.
type Discovery struct {
	basePath string
	kv       store.Store
	pairsMu  sync.RWMutex
	pairs    []*client.KVPair
	chans    []chan []*client.KVPair
	mu       sync.Mutex

	// -1 means it always retry to watch until zookeeper is ok, 0 means no retry.
	RetriesAfterWatchFailed int
	filter                  client.ServiceDiscoveryFilter
	stopCh                  chan struct{}
	closeOnce               sync.Once
}

// reconcileInterval 全量校验间隔。
//
// 🔴 watch 通道有两条静默失联路径:
//  1. libkv 的 watchLoop 内部出错时只退出内部循环、从不 close watch 通道——
//     本包 watch() 的 `<-c` 永远等不到,"chan is closed and will rewatch" 是死代码,
//     redis 一次抖动后 discovery 就永久冻结在旧列表上;
//  2. "最后一个节点过期/被删"的空列表通知被 libkv 的类型断言丢弃,死节点永不摘除
//     (单节点服务则永久挂死)。
//
// 定期全量 List 比对是唯一可靠的兜底:watch 只作加速,准确性以这里为准。
var reconcileInterval = 2 * time.Second

func (d *Discovery) reconcile() {
	ticker := time.NewTicker(reconcileInterval)
	defer ticker.Stop()
	for {
		select {
		case <-d.stopCh:
			return
		case <-ticker.C:
			ps, err := d.kv.List(d.basePath)
			if err != nil && !(AllowKeyNotFound && err == store.ErrKeyNotFound) {
				continue //本轮失败,下轮再试;保持上一份快照
			}
			prev := d.GetServices()
			pairs := d.setPairs(ps)
			//🔴 无变化不重推:rpcx xClient.watch 每收到一次推送都会 UpdateServer
			//重建 selectorNode,进程内累计的负载计数(Average)被 metadata 里的旧值
			//抹平——每 2s 盲推等于把最小负载选择每 2s 重新起跑
			if pairsEqual(prev, pairs) {
				continue
			}
			d.mu.Lock()
			for _, ch := range d.chans {
				select {
				case ch <- pairs:
				default:
				}
			}
			d.mu.Unlock()
		}
	}
}

// pairsEqual 按 key→value 集合比较两份快照(不比顺序,redis List 的返回序不稳定)
func pairsEqual(a, b []*client.KVPair) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]string, len(a))
	for _, p := range a {
		m[p.Key] = p.Value
	}
	for _, p := range b {
		if v, ok := m[p.Key]; !ok || v != p.Value {
			return false
		}
	}
	return true
}

// NewDiscovery returns a new Discovery.
func NewDiscovery(basePath string, servicePath string, redisAddr []string, options *store.Config) (*Discovery, error) {
	kv, err := libkv.NewStore(store.REDIS, redisAddr, options)
	if err != nil {
		log.Infof("cannot create store: %v", err)
		return nil, err
	}

	return NewDiscoveryStore(basePath+"/"+servicePath, kv)
}

// NewDiscoveryStore return a new Discovery with specified store.
func NewDiscoveryStore(basePath string, kv store.Store) (*Discovery, error) {
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = basePath[:len(basePath)-1]
	}

	d := &Discovery{basePath: basePath, kv: kv}
	d.stopCh = make(chan struct{})

	ps, err := kv.List(basePath)
	if err != nil && !(AllowKeyNotFound && err == store.ErrKeyNotFound) {
		log.Infof("cannot get services of from registry: %v, err: %v", basePath, err)
		return nil, err
	}
	d.setPairs(ps)
	d.RetriesAfterWatchFailed = -1
	go d.watch()
	go d.reconcile()
	return d, nil
}

// NewRedisDiscoveryTemplate returns a new Discovery template.
func NewRedisDiscoveryTemplate(basePath string, redisAddr []string, options *store.Config) (*Discovery, error) {
	if len(basePath) > 1 && strings.HasSuffix(basePath, "/") {
		basePath = basePath[:len(basePath)-1]
	}

	kv, err := libkv.NewStore(store.REDIS, redisAddr, options)
	if err != nil {
		log.Infof("cannot create store: %v", err)
		return nil, err
	}

	return NewDiscoveryStore(basePath, kv)
}

// Clone clones this ServiceDiscovery with new servicePath.
func (d *Discovery) Clone(servicePath string) (client.ServiceDiscovery, error) {
	return NewDiscoveryStore(d.basePath+"/"+servicePath, d.kv)
}

// SetFilter sets the filer.
func (d *Discovery) SetFilter(filter client.ServiceDiscoveryFilter) {
	d.filter = filter
}

// GetServices returns the servers
func (d *Discovery) GetServices() []*client.KVPair {
	d.pairsMu.RLock()
	defer d.pairsMu.RUnlock()

	return d.pairs
}

// WatchService returns a nil chan.
func (d *Discovery) WatchService() chan []*client.KVPair {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch := make(chan []*client.KVPair, 10)
	d.chans = append(d.chans, ch)
	return ch
}

func (d *Discovery) RemoveWatcher(ch chan []*client.KVPair) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var chans []chan []*client.KVPair
	for _, c := range d.chans {
		if c == ch {
			continue
		}

		chans = append(chans, c)
	}

	d.chans = chans
}

func (d *Discovery) watch() {
	defer func() {
		d.kv.Close()
	}()

	for {
		var err error
		var c <-chan []*store.KVPair
		var tempDelay time.Duration

		retry := d.RetriesAfterWatchFailed
		for d.RetriesAfterWatchFailed < 0 || retry >= 0 {
			c, err = d.kv.WatchTree(d.basePath, nil)
			if err != nil {
				if d.RetriesAfterWatchFailed > 0 {
					retry--
				}
				if tempDelay == 0 {
					tempDelay = 1 * time.Second
				} else {
					tempDelay *= 2
				}
				if n := 30 * time.Second; tempDelay > n {
					tempDelay = n
				}
				log.Warnf("can not watchtree (with retry %d, sleep %v): %s: %v", retry, tempDelay, d.basePath, err)
				time.Sleep(tempDelay)
				continue
			}
			break
		}

		if err != nil {
			log.Errorf("can't watch %s: %v", d.basePath, err)
			return
		}

	readChanges:
		for {
			select {
			case <-d.stopCh:
				log.Info("discovery has been closed")
				return
			case ps, ok := <-c:
				if !ok {
					break readChanges
				}
				var pairs []*client.KVPair // latest servers
				if ps == nil {
					d.pairsMu.Lock()
					d.pairs = pairs
					d.pairsMu.Unlock()
					continue
				}
				pairs = d.setPairs(ps)
				d.mu.Lock()
				for _, ch := range d.chans {
					// 非阻塞发送，满则丢弃。避免为每个 watcher 起 goroutine + time.After 泄漏 timer
					select {
					case ch <- pairs:
					default:
						log.Warn("chan is full and new change has been dropped")
					}
				}
				d.mu.Unlock()
			}
		}

		log.Warn("chan is closed and will rewatch")
	}
}

func (d *Discovery) Close() {
	d.closeOnce.Do(func() {
		close(d.stopCh)
	})
}

func (d *Discovery) prefix(key string) (prefix string) {
	if strings.HasPrefix(key, "/") {
		if strings.HasPrefix(d.basePath, "/") {
			prefix = d.basePath + "/"
		} else {
			prefix = "/" + d.basePath + "/"
		}
	} else {
		if strings.HasPrefix(d.basePath, "/") {
			prefix = d.basePath[1:] + "/"
		} else {
			prefix = d.basePath + "/"
		}
	}
	return
}

func (d *Discovery) setPairs(ps []*store.KVPair) []*client.KVPair {
	pairs := make([]*client.KVPair, 0, len(ps))
	var prefix string
	for _, p := range ps {
		if prefix == "" {
			prefix = d.prefix(p.Key)
		}
		if p.Key == prefix[:len(prefix)-1] {
			continue
		}
		k := strings.TrimPrefix(p.Key, prefix)
		pair := &client.KVPair{Key: k, Value: string(p.Value)}
		if d.filter != nil && !d.filter(pair) {
			continue
		}
		pairs = append(pairs, pair)
	}
	d.pairsMu.Lock()
	d.pairs = pairs
	d.pairsMu.Unlock()
	return pairs
}
