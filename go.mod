module github.com/hwcer/cosrpc

go 1.16

replace (
github.com/hwcer/cosgo v0.0.1 => ../cosgo
github.com/hwcer/registry v0.0.1 => ../registry
)

require (
	github.com/hwcer/cosgo v0.0.1
	github.com/hwcer/logger v0.0.1
	github.com/hwcer/registry v0.0.1
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/rpcxio/libkv v0.5.1
	github.com/smallnest/rpcx v1.7.8
)
