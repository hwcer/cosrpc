package server

import (
	"errors"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosrpc"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
)

// RegistryMethod 定义服务注册的方法名
const RegistryMethod = "RPCX"

// Caller 定义服务调用接口
// 用于处理 RPC 请求并返回结果
type Caller interface {
	Caller(c *cosrpc.Context, node *registry.Node) interface{}
}

// Register 定义服务注册接口
// 用于管理服务的注册、启动和停止
type Register interface {
	Stop() error
	Start() error
	Register(name string, rcvr interface{}, metadata string) (err error)
}

// New 创建并返回一个新的 Server 实例
// 初始化内部的 rpcx Server 和服务注册表
func New() *Server {
	r := &Server{}
	r.Server = server.NewServer()
	r.Registry = registry.New()
	r.Server.DisableHTTPGateway = true
	return r
}

// Server 是 cosrpc 服务器的核心结构
// 封装了 rpcx Server 并提供了服务注册和管理功能
type Server struct {
	*server.Server                    // 内嵌的 rpcx Server
	started        int32              // 服务器启动状态，0 未启动，1 已启动
	register       Register           // 服务注册器
	Registry       *registry.Registry // 服务注册表
}

// Caller 处理 RPC 请求的入口方法
// 1. 从 node 中获取 Handler
// 2. 创建 cosrpc Context
// 3. 调用 Handler.Caller 处理请求
// 4. 序列化响应并写入客户端
func (xs *Server) Caller(sc cosrpc.ICtx, node *registry.Node) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Alert("rpcx server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()

	handler, ok := node.Handler().(*Handler)
	if !ok {
		return errors.New("handler unknown")
	}
	c := cosrpc.NewContext(sc)
	var reply any
	reply, err = handler.Caller(node, c)
	if err != nil {
		return
	}
	var data []byte
	if data, err = handler.Marshal(c, reply); err == nil {
		return c.Write(data)
	}
	return
}

// Service 创建并返回一个新的服务
// 1. 创建一个新的 Handler
// 2. 在注册表中创建服务
// 3. 设置服务方法
// 4. 应用传入的处理器
func (xs *Server) Service(name string, handlers ...any) *registry.Service {
	handler := &Handler{}
	service := xs.Registry.Service(name, handler)
	service.SetMethods([]string{RegistryMethod})
	for _, i := range handlers {
		handler.Use(i)
	}
	return service
}

// startServer 启动 RPC 服务器
// 1. 调用 rpcx Server 的 Serve 方法
// 2. 设置 1 秒超时，防止阻塞
func (xs *Server) startServer(network, address string) (err error) {
	err = scc.Timeout(time.Second, func() error {
		return xs.Server.Serve(network, address)
	})
	if errors.Is(scc.ErrorTimeout, err) {
		err = nil
	}
	return
}

// startRegister 启动服务注册
// 1. 检查默认注册器是否存在
// 2. 创建注册器实例
// 3. 收集服务信息
// 4. 注册服务
// 5. 启动注册器
func (xs *Server) startRegister() (err error) {
	if defaultRegister == nil {
		logger.Alert("register is nil,Can only run in standalone mode")
		return nil
	}
	if xs.register, err = defaultRegister(); err != nil {
		return err
	}
	// 注册服务,实现 rpcxServiceHandlerMetadata 才具有服务发现功能
	service := map[string]string{}
	xs.Registry.Range(func(s *registry.Service) bool {
		name := strings.TrimPrefix(s.Name(), "/")
		service[name] = Metadata.Get(name)
		return true
	})
	if len(service) == 0 {
		return
	}
	for name, meta := range service {
		if err = xs.register.Register(name, nil, meta); err != nil {
			return err
		}
	}
	if err = xs.register.Start(); err != nil {
		return
	}
	xs.Server.Plugins.Add(xs.Registry)
	return
}

// parseServiceName 解析服务名称
// 将服务名称解析为服务路径和服务方法
func (xs *Server) parseServiceName(name string) (servicePath string, serviceMethod string) {
	name = strings.TrimPrefix(name, "/")
	i := strings.Index(name, "/")
	servicePath = name[:i]
	serviceMethod = name[i:]
	return
}

// Start 启动服务器
// 1. 检查服务注册表是否为空
// 2. 原子操作检查并设置启动状态
// 3. 获取服务器地址
// 4. 为每个服务节点添加处理器
// 5. 启动服务器
// 6. 启动服务注册
func (xs *Server) Start() (err error) {
	if xs.Registry.Len() == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&xs.started, 0, 1) {
		return
	}
	address := cosrpc.Address()
	// 启动服务
	xs.Registry.Nodes(func(node *registry.Node) (r bool) {
		servicePath, serviceMethod := xs.parseServiceName(node.Name())
		var handler = func(c *server.Context) error {
			return xs.Caller(c, node)
		}
		xs.Server.AddHandler(servicePath, serviceMethod, handler)
		return true
	})

	err = address.Handle(func(network, address string) error {
		return xs.startServer(network, address)
	})
	if err != nil {
		return
	}

	if err = xs.startRegister(); err != nil {
		return
	}
	logger.Trace("rpc server started:%v", address.String())
	return
}

// Close 关闭服务器
// 1. 原子操作检查并设置启动状态
// 2. 关闭 rpcx Server
// 3. 停止服务注册
func (xs *Server) Close() (err error) {
	if !atomic.CompareAndSwapInt32(&xs.started, 1, 0) {
		return
	}
	if err = xs.Server.Shutdown(nil); err != nil {
		return
	}
	if xs.register != nil {
		err = xs.register.Stop()
	}
	return
}
