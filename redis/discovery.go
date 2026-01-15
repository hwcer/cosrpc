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
					ch := ch
					go func() {
						defer func() {
							recover()
						}()

						select {
						case ch <- pairs:
						case <-time.After(time.Minute):
							log.Warn("chan is full and new change has been dropped")
						}
					}()
				}
				d.mu.Unlock()
			}
		}

		log.Warn("chan is closed and will rewatch")
	}
}

func (d *Discovery) Close() {
	close(d.stopCh)
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
