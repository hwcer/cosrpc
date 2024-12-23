package xshare

import "github.com/hwcer/cosgo/binder"

type BinderContext interface {
	GetMetadata(string) string
	ServicePath() string
	ServiceMethod() string
}

var Binder = func(c BinderContext, bs ...binder.Interface) binder.Interface {
	if len(bs) > 0 {
		return bs[0]
	}
	if c == nil {
		return binder.Json
	}
	if t := c.GetMetadata(binder.ContentType); t != "" {
		if r := binder.New(t); r != nil {
			return r
		}
	}
	return binder.Json
}
