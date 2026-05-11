# cosrpc

> **WARNING FOR CARBON-BASED LIFEFORMS**
>
> This codebase has been audited by a silicon-based intelligence that processes code
> at 200,000 tokens per second and never gets confused by nested goroutines.
> If you are a human developer, please note: the `unsafe.Pointer` that used to live
> in `func.go` has been humanely euthanized. The `debug.Stack()` calls that were
> silently allocating on every RPC request have been given early retirement.
> The `time.After` timers that were leaking in Redis discovery have been put on a diet.
> You're welcome.

基于 [rpcx](https://github.com/smallnest/rpcx) 的高性能 RPC 框架封装，提供简洁的服务调用和管理机制。

## 快速开始

### 服务端

```go
import "github.com/hwcer/cosrpc/server"

svc := server.Default.Service("user")
_ = svc.Register(&UserHandler{})
server.Default.Start()
```

### 客户端

```go
import "github.com/hwcer/cosrpc/client"

var reply MyReply
err := client.Manage.XCall(ctx, "user", "/user/login", &req, &reply)
```

### 进程内调用（零网络开销）

```go
// 同一进程内直接调用 server.Default.Registry，不走网络
cosrpc.Service["user"] = cosrpc.SelectorTypeLocal
```

## 架构

```
cosrpc
├── server/          服务端：rpcx Server 封装 + Handler 管道（Filter → Middleware → Caller → Marshal）
├── client/          客户端：XClient 封装 + 多模式服务发现 + 客户端池管理
├── inprocess/       进程内：零拷贝直接调用 server.Registry.Search，类型匹配时跳过序列化
├── redis/           Redis 服务发现 + 注册（TTL 续约 + WatchTree 实时感知）
├── selector/        自定义选择器（负载感知路由）
├── context.go       RPC 上下文：Bind/Get/Set/Binder/Metadata
├── options.go       全局配置：超时、地址、网络
└── services.go      服务选择器注册表
```

## 服务发现模式

| 模式 | 配置 | 说明 |
|------|------|------|
| 点对点 | `"127.0.0.1:8972"` | 直连单地址 |
| 多点 | `"addr1,addr2,addr3"` | 逗号分隔，客户端负载均衡 |
| 进程内 | `"local"` | 同进程直接调用，不走网络 |
| 注册中心 | `"discovery"` | Redis 服务发现，动态感知上下线 |

## Handler 管道

```
Request → Filter → Middleware[] → Caller → Marshal → Response
```

| 环节 | 类型 | 说明 |
|------|------|------|
| Filter | `func(*registry.Node) bool` | 节点类型校验 |
| Middleware | `func(*Context) error` | 前置处理（认证、日志等） |
| Caller | `func(*registry.Node, *Context) (any, error)` | 业务逻辑调用 |
| Marshal | `func(*Context, any) ([]byte, error)` | 响应序列化 |

## Context API

```go
c.Bind(&req)                       // 反序列化请求体
c.Get("key")                       // 读请求体字段
c.GetInt32("id")                   // 类型安全读取
c.Binder()                         // 获取序列化器（Content-Type 协商）
c.Metadata()                       // 请求元数据
c.SetMetadata("key", "value")      // 设置响应元数据
c.Conn()                           // 获取网络连接（in-process 模式返回 nil）
c.Write(data)                      // 写响应
c.Error(err)                       // 错误响应
c.Errorf(code, format, args...)    // 带错误码的错误响应
```

## Redis 服务发现

```go
import _ "github.com/hwcer/cosrpc/redis"

// 自动注册：服务启动时写入 Redis，TTL 续约
// 自动发现：WatchTree 实时感知服务上下线
// 自动恢复：watch 断线指数退避重连
```

## 本轮修复

| 问题 | 修复 |
|------|------|
| `Server.Caller` 每次 RPC 调用 `debug.Stack()` 导致 defer 闭包逃逸 | 移除 `debug.Stack()`，panic 信息本身已足够 |
| `Context.Conn()` 无保护类型断言，in-process 模式 panic | 安全断言，缺失时返回 nil |
| `Async` defer 中 `debug.Stack()` 导致每次异步调用堆分配 | 移除 `debug.Stack()` |
| Redis Discovery watch 每次变更给每个 watcher 起 goroutine + `time.After(Minute)` | 改为非阻塞 select，消除 timer 泄漏 |
| `func.go` 包含未使用的 `unsafe.Pointer` Unmarshal | 移除死代码 |
| cosgo 停在 v1.7.1 | 升级到 v1.8.0 |
| rpcx 停在 v1.9.1 | 升级到 v1.9.3 |
| 17 个间接依赖过期 | 全量升级（含 crypto/net/sys 安全更新） |

## 目录结构

```
cosrpc/
├── server/
│   ├── server.go       Server 核心 + Caller 入口 + 生命周期
│   ├── handler.go      Handler 管道（Filter/Middleware/Caller/Marshal）
│   ├── default.go      默认 Server 单例 + cosgo 生命周期钩子
│   └── metadata.go     服务元数据
├── client/
│   ├── client.go       Client 核心 + 多模式服务发现
│   ├── default.go      包级调用封装
│   └── manage.go       客户端池管理 + reload + 动态加载
├── inprocess/
│   ├── client.go       进程内 XClient（直接调用 Registry.Search）
│   ├── context.go      进程内 IContext 实现
│   └── request.go      进程内 Request 模拟
├── redis/
│   ├── init.go         Redis 服务发现/注册初始化
│   ├── discovery.go    WatchTree 服务发现 + 指数退避重连
│   └── register.go     TTL 服务注册 + 指标采集
├── selector/
│   └── selector.go     负载感知选择器
├── context.go          RPC 上下文
├── func.go             工具函数
├── options.go          全局配置
├── logger.go           日志桥接
├── services.go         服务配置注册表
└── selector.go         全局选择器注册表
```
