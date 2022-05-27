package cosrpc

import (
	"github.com/smallnest/rpcx/server"
	"reflect"
)

type Context struct {
	*server.Context
}

var typeOfContext = reflect.TypeOf((*Context)(nil)).Elem()
