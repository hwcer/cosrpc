package redis

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/rpcxio/libkv"
	"github.com/rpcxio/libkv/store"
	"github.com/rpcxio/libkv/store/redis"
	"github.com/smallnest/rpcx/log"
)

func init() {
	redis.Register()
}

// Register implements redis registry.
type Register struct {
	// service address, for example, tcp@127.0.0.1:8972, quic@127.0.0.1:1234
	ServiceAddress string
	// redis addresses
	RedisServers []string
	// base path for rpcx server, for example com/example/rpcx
	BasePath string
	Metrics  metrics.Registry
	// Registered services
	Services       []string
	metasLock      sync.RWMutex
	metas          map[string]string
	UpdateInterval time.Duration

	Options *store.Config
	kv      store.Store

	dying    chan struct{}
	done     chan struct{}
	stopOnce sync.Once //Stop 防重入:close(dying) 二次调用会 panic
}

// Start starts to connect redis cluster
func (p *Register) Start() error {
	if p.done == nil {
		p.done = make(chan struct{})
	}
	if p.dying == nil {
		p.dying = make(chan struct{})
	}

	if p.kv == nil {
		kv, err := libkv.NewStore(store.REDIS, p.RedisServers, p.Options)
		if err != nil {
			log.Errorf("cannot create redis registry: %v", err)
			close(p.done)
			return err
		}
		p.kv = kv
	}

	err := p.kv.Put(p.BasePath, []byte("rpcx_path"), &store.WriteOptions{IsDir: true})
	if err != nil && !strings.Contains(err.Error(), "Not a file") {
		log.Errorf("cannot create redis path %s: %v", p.BasePath, err)
		close(p.done)
		return err
	}

	if p.UpdateInterval > 0 {
		go func() {
			ticker := time.NewTicker(p.UpdateInterval)

			defer ticker.Stop()
			//kv 的 Close 移到 Stop:旧实现在这里无条件关闭,Stop 删节点时连接已断,
			//Delete 全部失败,节点只能等 TTL 过期

			// refresh service TTL
			for {
				select {
				case <-p.dying:
					close(p.done)
					return
				case <-ticker.C:
					extra := make(map[string]string)
					if p.Metrics != nil {
						extra["calls"] = fmt.Sprintf("%.2f", metrics.GetOrRegisterMeter("calls", p.Metrics).RateMean())
						extra["connections"] = fmt.Sprintf("%.2f", metrics.GetOrRegisterMeter("connections", p.Metrics).RateMean())
					}
					//set this same metrics for all services at this server
					p.metasLock.RLock()
					services := p.Services
					p.metasLock.RUnlock()
					for _, name := range services {
						nodePath := fmt.Sprintf("%s/%s/%s", p.BasePath, name, p.ServiceAddress)
						if p.Metrics == nil {
							//🔴 无指标合并需求时免 GET 直接 SET 续期:TTL 续期本身只需
							//覆盖 TTL,旧实现每 tick 每服务一次 GET+ParseQuery+重编码
							//纯属浪费,且每次 SET 都触发 keyspace 事件放大 watch 流量
							err := p.kv.Put(nodePath, []byte(p.metasGet(name)), &store.WriteOptions{TTL: p.UpdateInterval * 2})
							if err != nil {
								log.Errorf("cannot renew redis path %s: %v", nodePath, err)
							}
							continue
						}
						kvPair, err := p.kv.Get(nodePath)
						if err != nil {
							log.Infof("can't get data of node: %s, because of %v", nodePath, err.Error())

							err = p.kv.Put(nodePath, []byte(p.metasGet(name)), &store.WriteOptions{TTL: p.UpdateInterval * 2})
							if err != nil {
								log.Errorf("cannot re-create redis path %s: %v", nodePath, err)
							}

						} else {
							v, _ := url.ParseQuery(string(kvPair.Value))
							for key, value := range extra {
								v.Set(key, value)
							}
							_ = p.kv.Put(nodePath, []byte(v.Encode()), &store.WriteOptions{TTL: p.UpdateInterval * 2})
						}
					}
				}
			}
		}()
	}

	return nil
}

// Stop unregister all services.
//
// 🔴 顺序必须是"先停续期协程、再删节点、最后关连接":旧实现先 Delete 再 close(dying),
// 间隙内 ticker 触发时续期协程 Get 失败会走 metas 重建逻辑把已下线节点重新写入 redis,
// 客户端继续路由到已停止的服务直至 TTL 过期(僵尸注册)。
// Start 从未执行(Register 中途失败,如 redis 不可用)时 done/dying 为 nil,
// 旧实现的 <-p.done 对 nil channel 永久阻塞,进程无法优雅退出。
func (p *Register) Stop() error {
	p.stopOnce.Do(func() {
		//1. 先停续期协程并等它退出(nil 防护:Register 失败未 Start 时两者皆 nil)
		if p.dying != nil {
			close(p.dying)
		}
		if p.done != nil {
			<-p.done
		}

		//2. 再删服务节点
		if p.kv != nil {
			p.metasLock.RLock()
			services := p.Services
			p.metasLock.RUnlock()
			for _, name := range services {
				nodePath := fmt.Sprintf("%s/%s/%s", p.BasePath, name, p.ServiceAddress)
				exist, err := p.kv.Exists(nodePath)
				if err != nil {
					log.Errorf("cannot delete path %s: %v", nodePath, err)
					continue
				}
				if exist {
					_ = p.kv.Delete(nodePath)
					log.Infof("delete path %s", nodePath)
				}
			}
		}

		//3. 最后关连接(原续期协程的 defer p.kv.Close() 已移除)
		if p.kv != nil {
			p.kv.Close()
		}
	})
	return nil
}

// metasGet 读指定服务的注册元数据(调用方自行决定锁范围,这里统一持读锁)
func (p *Register) metasGet(name string) string {
	p.metasLock.RLock()
	defer p.metasLock.RUnlock()
	return p.metas[name]
}

// HandleConnAccept handles connections from clients
func (p *Register) HandleConnAccept(conn net.Conn) (net.Conn, bool) {
	if p.Metrics != nil {
		metrics.GetOrRegisterMeter("connections", p.Metrics).Mark(1)
	}
	return conn, true
}

// PreCall handles rpc call from clients
func (p *Register) PreCall(_ context.Context, _, _ string, args any) (any, error) {
	if p.Metrics != nil {
		metrics.GetOrRegisterMeter("calls", p.Metrics).Mark(1)
	}
	return args, nil
}

// Register handles registering event.
// this service is registered at BASE/serviceName/thisIpAddress node
func (p *Register) Register(name string, rcvr any, metadata string) (err error) {
	if strings.TrimSpace(name) == "" {
		err = errors.New("Register service `name` can't be empty")
		return
	}

	if p.kv == nil {
		redis.Register()
		kv, err := libkv.NewStore(store.REDIS, p.RedisServers, p.Options)
		if err != nil {
			log.Errorf("cannot create redis registry: %v", err)
			return err
		}
		p.kv = kv
	}

	err = p.kv.Put(p.BasePath, []byte("rpcx_path"), &store.WriteOptions{IsDir: true})
	if err != nil && !strings.Contains(err.Error(), "Not a file") {
		log.Errorf("cannot create redis path %s: %v", p.BasePath, err)
		return err
	}

	nodePath := fmt.Sprintf("%s/%s", p.BasePath, name)
	err = p.kv.Put(nodePath, []byte(name), &store.WriteOptions{IsDir: true})
	if err != nil && !strings.Contains(err.Error(), "Not a file") {
		log.Errorf("cannot create redis path %s: %v", nodePath, err)
		return err
	}

	nodePath = fmt.Sprintf("%s/%s/%s", p.BasePath, name, p.ServiceAddress)
	err = p.kv.Put(nodePath, []byte(metadata), &store.WriteOptions{TTL: p.UpdateInterval * 2})
	if err != nil {
		log.Errorf("cannot create redis path %s: %v", nodePath, err)
		return err
	}

	//Services 与 metas 同锁:续期协程/Stop 读 Services 与 Register/Unregister 写并发
	p.metasLock.Lock()
	p.Services = append(p.Services, name)
	if p.metas == nil {
		p.metas = make(map[string]string)
	}
	p.metas[name] = metadata
	p.metasLock.Unlock()
	return
}

func (p *Register) Unregister(name string) (err error) {
	if len(p.Services) == 0 { //仅启动期单线程调用,无需加锁
		return nil
	}

	if strings.TrimSpace(name) == "" {
		err = errors.New("Register service `name` can't be empty")
		return
	}

	if p.kv == nil {
		redis.Register()
		kv, err := libkv.NewStore(store.REDIS, p.RedisServers, p.Options)
		if err != nil {
			log.Errorf("cannot create redis registry: %v", err)
			return err
		}
		p.kv = kv
	}

	err = p.kv.Put(p.BasePath, []byte("rpcx_path"), &store.WriteOptions{IsDir: true})
	if err != nil && !strings.Contains(err.Error(), "Not a file") {
		log.Errorf("cannot create redis path %s: %v", p.BasePath, err)
		return err
	}

	nodePath := fmt.Sprintf("%s/%s", p.BasePath, name)
	err = p.kv.Put(nodePath, []byte(name), &store.WriteOptions{IsDir: true})
	if err != nil && !strings.Contains(err.Error(), "Not a file") {
		log.Errorf("cannot create redis path %s: %v", nodePath, err)
		return err
	}

	nodePath = fmt.Sprintf("%s/%s/%s", p.BasePath, name, p.ServiceAddress)

	err = p.kv.Delete(nodePath)
	if err != nil {
		log.Errorf("cannot remove redis path %s: %v", nodePath, err)
		return err
	}

	p.metasLock.Lock()
	services := make([]string, 0, len(p.Services))
	for _, s := range p.Services {
		if s != name {
			services = append(services, s)
		}
	}
	p.Services = services
	if p.metas == nil {
		p.metas = make(map[string]string)
	}
	delete(p.metas, name)
	p.metasLock.Unlock()
	return
}
