module github.com/hwcer/cosrpc

go 1.16

replace (
	github.com/hwcer/cosgo v0.0.0 => ../cosgo
	github.com/hwcer/registry v0.0.0 => ../registry
)

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/hwcer/cosgo v0.0.0
	github.com/hwcer/registry v0.0.0
	github.com/nacos-group/nacos-sdk-go v1.1.1
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/rpcxio/libkv v0.5.1-0.20210420120011-1fceaedca8a5
	github.com/rpcxio/rpcx-etcd v0.2.0
	github.com/smallnest/rpcx v1.7.4
	github.com/stretchr/testify v1.7.1
	go.etcd.io/etcd/client/v2 v2.305.4
	go.etcd.io/etcd/client/v3 v3.5.4
)

require (
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1 // indirect
)
