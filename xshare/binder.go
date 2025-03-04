package xshare

import (
	"context"
	"github.com/hwcer/cosgo/binder"
	"github.com/smallnest/rpcx/share"
)

type BinderMod int8

const (
	BinderModReq BinderMod = iota
	BinderModRes
)

func GetBinderFromContext(ctx context.Context, mod BinderMod) (r binder.Binder) {
	if ctx == nil {
		return Binder
	}
	if i := ctx.Value(share.ReqMetaDataKey); i != nil {
		if meta, ok := i.(map[string]string); ok {
			r = GetBinderFromMetadata(meta, mod)
		}
	}
	if r == nil {
		r = Binder
	}
	return
}

func GetBinderFromMetadata(meta map[string]string, mod BinderMod) (r binder.Binder) {
	var k string
	if mod == BinderModReq {
		k = binder.HeaderContentType
	} else {
		k = binder.HeaderAccept
	}
	if ct := meta[k]; ct != "" {
		r = binder.New(ct)
	}
	if r == nil && mod == BinderModRes {
		r = GetBinderFromMetadata(meta, BinderModReq) //保持和请求时一致
	}
	if r == nil {
		r = Binder
	}
	return
}
