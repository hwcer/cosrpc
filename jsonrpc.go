package cosrpc

import "github.com/hwcer/cosrpc/jsonrpc"

type jsonrpcHandle func(c *Context, args []*jsonrpc.Args) ([]*jsonrpc.Reply, error)
