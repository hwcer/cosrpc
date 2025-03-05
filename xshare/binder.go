package xshare

import (
	"context"
	"github.com/hwcer/cosgo/binder"
	"github.com/smallnest/rpcx/share"
)

func GetBinderFromContext(ctx context.Context, mod binder.ContentTypeMod) (r binder.Binder) {
	if ctx == nil {
		return binder.Default
	}
	if i := ctx.Value(share.ReqMetaDataKey); i != nil {
		if meta, ok := i.(map[string]string); ok {
			r = binder.GetContentType(meta, mod)
		}
	}
	if r == nil {
		r = binder.Default
	}
	return
}
