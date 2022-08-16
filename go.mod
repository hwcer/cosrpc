module github.com/hwcer/cosrpc

go 1.16

replace (
	github.com/hwcer/cosgo v0.0.0 => ../cosgo
	github.com/hwcer/registry v0.0.0 => ../registry
)

require (
	github.com/hwcer/cosgo v0.0.0
	github.com/hwcer/registry v0.0.0
	github.com/smallnest/rpcx v1.7.8
)
