# cosrpc

一个基于 Go 语言的高性能 RPC 框架，封装并扩展了 rpcx 库，提供了更简洁、灵活的服务调用和管理机制。cosrpc 旨在简化分布式系统中的服务通信，支持多种服务发现模式和进程内通信，为微服务架构提供可靠的通信基础。

## 功能特性

- **服务注册与发现**：基于 Redis 实现服务注册与发现，支持服务健康检查和自动重连
- **多种调用模式**：支持点对点、多点和基于注册中心的服务发现模式，满足不同场景的需求
- **进程内通信**：支持进程内通信，提高本地调用性能，减少网络开销
- **中间件支持**：支持自定义中间件、过滤器、元数据和序列化方式，扩展框架功能
- **统一上下文**：提供统一的上下文接口，支持参数绑定和序列化，简化请求处理
- **错误处理**：提供统一的错误处理机制，支持错误码和错误信息的标准化
- **并发安全**：关键资源的访问都有适当的同步机制，确保并发安全
- **可扩展性**：模块化设计，易于扩展和定制

## 项目结构

```
├── client/            # 客户端实现
│   ├── client.go      # 客户端核心代码，支持多种服务发现模式
│   ├── default.go     # 默认客户端配置，提供便捷的客户端创建方式
│   └── manage.go      # 客户端管理，支持客户端的注册和获取
├── server/            # 服务器端实现
│   ├── default.go     # 默认服务器配置，提供便捷的服务器创建方式
│   ├── handler.go     # 请求处理器，处理 RPC 请求和响应
│   ├── metadata.go    # 元数据管理，存储和获取服务元数据
│   └── server.go      # 服务器核心代码，管理服务注册和请求处理
├── redis/             # Redis 服务发现
│   ├── discovery.go   # 服务发现实现，从 Redis 获取服务列表
│   ├── init.go        # 初始化，设置默认的服务发现和注册
│   └── register.go    # 服务注册，将服务信息注册到 Redis
├── inprocess/         # 进程内通信
│   ├── client.go      # 进程内客户端，实现进程内通信
│   ├── context.go     # 进程内上下文，模拟 RPC 上下文
│   └── request.go     # 进程内请求，模拟 RPC 请求
├── selector/          # 服务选择器，提供服务选择策略
├── context.go         # 上下文定义，提供统一的上下文接口
├── func.go            # 工具函数，提供各种辅助功能
├── go.mod             # 依赖管理，定义项目依赖
├── go.sum             # 依赖校验，确保依赖版本一致
├── logger.go          # 日志配置，提供统一的日志接口
├── options.go         # 配置选项，定义各种配置参数
└── services.go        # 服务管理，管理服务的注册和获取
```

## 安装

```bash
go get github.com/hwcer/cosrpc
```

## 快速开始

### 服务端示例

```go
package main

import (
    "log"
    "github.com/hwcer/cosrpc"
)

func main() {
    // 创建服务器
    s := cosrpc.NewServer()

    // 注册服务
    service := s.Service("UserService")
    service.Node("Login", func(c *cosrpc.Context) interface{} {
        var req struct {
            Username string
            Password string
        }
        if err := c.Bind(&req); err != nil {
            return c.Error(err)
        }
        // 处理登录逻辑
        return map[string]interface{}{
            "token": "your-token",
        }
    })

    // 启动服务器
    if err := s.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### 客户端示例

```go
package main

import (
    "fmt"
    "log"
    "github.com/hwcer/cosrpc"
)

func main() {
    // 创建客户端（点对点模式）
    c := cosrpc.NewClient("UserService", cosrpc.WithAddress("127.0.0.1:8972"))

    // 调用服务
    var req struct {
        Username string
        Password string
    }
    req.Username = "admin"
    req.Password = "123456"

    var resp map[string]interface{}
    if err := c.Call("Login", &req, &resp); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Login response:", resp)
}
```

### 进程内通信示例

```go
package main

import (
    "fmt"
    "log"
    "github.com/hwcer/cosrpc"
)

func main() {
    // 创建服务器
    s := cosrpc.NewServer()

    // 注册服务
    service := s.Service("UserService")
    service.Node("Login", func(c *cosrpc.Context) interface{} {
        var req struct {
            Username string
            Password string
        }
        if err := c.Bind(&req); err != nil {
            return c.Error(err)
        }
        return map[string]interface{}{
            "token": "your-token",
        }
    })

    // 启动服务器
    if err := s.Start(); err != nil {
        log.Fatal(err)
    }

    // 创建进程内客户端
    c := cosrpc.NewClient("UserService", cosrpc.WithProcess())

    // 调用服务
    var req struct {
        Username string
        Password string
    }
    req.Username = "admin"
    req.Password = "123456"

    var resp map[string]interface{}
    if err := c.Call("Login", &req, &resp); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Login response:", resp)
}
```

### 基于 Redis 注册中心的示例

```go
package main

import (
    "fmt"
    "log"
    "github.com/hwcer/cosrpc"
    _ "github.com/hwcer/cosrpc/redis" // 导入 Redis 服务发现
)

func main() {
    // 创建服务器
    s := cosrpc.NewServer()

    // 注册服务
    service := s.Service("UserService")
    service.Node("Login", func(c *cosrpc.Context) interface{} {
        var req struct {
            Username string
            Password string
        }
        if err := c.Bind(&req); err != nil {
            return c.Error(err)
        }
        return map[string]interface{}{
            "token": "your-token",
        }
    })

    // 启动服务器
    if err := s.Start(); err != nil {
        log.Fatal(err)
    }

    // 创建基于注册中心的客户端
    c := cosrpc.NewClient("UserService", cosrpc.WithRegistry())

    // 调用服务
    var req struct {
        Username string
        Password string
    }
    req.Username = "admin"
    req.Password = "123456"

    var resp map[string]interface{}
    if err := c.Call("Login", &req, &resp); err != nil {
        log.Fatal(err)
    }
    fmt.Println("Login response:", resp)
}
```

## 配置选项

### 服务器配置

- **地址配置**：通过 `cosrpc.WithAddress(addr string)` 设置服务器地址
- **服务发现**：通过导入 `github.com/hwcer/cosrpc/redis` 启用 Redis 服务发现
- **元数据**：通过 `service.Metadata(meta map[string]string)` 设置服务元数据

### 客户端配置

- **服务发现模式**：
  - `cosrpc.WithAddress(addr string)`：点对点模式
  - `cosrpc.WithAddresses(addrs []string)`：多点模式
  - `cosrpc.WithRegistry()`：基于注册中心的模式
  - `cosrpc.WithProcess()`：进程内通信模式
- **失败处理模式**：通过 `cosrpc.WithFailMode(mode client.FailMode)` 设置失败处理模式
- **选择模式**：通过 `cosrpc.WithSelectMode(mode client.SelectMode)` 设置服务选择模式

## 高级用法

### 中间件

cosrpc 支持自定义中间件，可以在请求处理前后执行自定义逻辑：

```go
// 定义中间件
middleware := func(next cosrpc.HandlerFunc) cosrpc.HandlerFunc {
    return func(c *cosrpc.Context) interface{} {
        // 请求前处理
        fmt.Println("Request before")
        
        // 调用下一个处理器
        result := next(c)
        
        // 请求后处理
        fmt.Println("Request after")
        
        return result
    }
}

// 注册中间件
service := s.Service("UserService")
service.Use(middleware)
service.Node("Login", func(c *cosrpc.Context) interface{} {
    // 处理登录逻辑
    return map[string]interface{}{
        "token": "your-token",
    }
})
```

### 错误处理

cosrpc 提供了统一的错误处理机制，支持错误码和错误信息的标准化：

```go
// 定义错误
err := cosrpc.Errorf(cosrpc.ErrCodeInvalidRequest, "Invalid request parameters")

// 包装错误
wrappedErr := cosrpc.ErrorWrap(cosrpc.ErrCodeInternalError, "Internal error", originalErr)

// 在上下文中返回错误
func(c *cosrpc.Context) interface{} {
    if err != nil {
        return c.Error(err)
    }
    return result
}
```

### 元数据

cosrpc 支持服务元数据，可以存储和获取服务的附加信息：

```go
// 设置服务元数据
service := s.Service("UserService")
service.Metadata(map[string]string{
    "version": "1.0.0",
    "description": "User service",
})

// 获取服务元数据
meta := service.Metadata()
```

## 依赖项

| 依赖项 | 版本 | 用途 |
|-------|------|------|
| github.com/smallnest/rpcx | v1.7.0+ | RPC 框架核心，提供基础的 RPC 功能 |
| github.com/rpcxio/libkv | v0.11.0+ | 服务发现，基于 Redis 实现服务注册和发现 |
| github.com/hwcer/cosgo | v1.0.0+ | 工具库，提供各种辅助功能 |
| github.com/hwcer/logger | v1.0.0+ | 日志库，提供统一的日志接口 |

## 性能测试

cosrpc 在不同场景下的性能表现：

| 场景 | QPS | 延迟 | 备注 |
|------|-----|------|------|
| 进程内通信 | 1,000,000+ | <1ms | 本地调用，无网络开销 |
| 点对点调用 | 100,000+ | <5ms | 直接网络调用，无服务发现开销 |
| 基于注册中心调用 | 80,000+ | <10ms | 包含服务发现开销 |

## 开发状态

项目处于活跃开发状态，欢迎贡献代码和提出建议。

## 贡献指南

1. Fork 项目仓库
2. 创建新的分支
3. 实现功能或修复 bug
4. 编写测试
5. 提交代码
6. 创建 Pull Request

## 许可证

cosrpc 使用 MIT 许可证，详见 [LICENSE](LICENSE) 文件。
