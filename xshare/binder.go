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

const (
	MetadataHeaderContentTypeRequest  = "_ctq" //客户端请求使用的序列化方式
	MetadataHeaderContentTypeResponse = "_cts" //客户端可以接受的序列化方式,默认不设置和请求序列化一样
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

func GetBinderFromMetadata(meta Metadata, mod BinderMod) (r binder.Binder) {
	var k string
	if mod == BinderModReq {
		k = MetadataHeaderContentTypeRequest
	} else {
		k = MetadataHeaderContentTypeResponse
	}
	if ct := meta.Get(k); ct != "" {
		r = binder.New(ct)
	}
	if r == nil {
		if mod == BinderModRes {
			r = GetBinderFromMetadata(meta, BinderModReq) //保持和请求时一致
		} else {
			r = Binder
		}
	}
	return
}
