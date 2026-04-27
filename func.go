package cosrpc

import (
	"context"
	"reflect"

	"github.com/hwcer/cosgo/binder"
	"github.com/smallnest/rpcx/share"
)

func GetBinderFromContext(ctx context.Context, cts ...string) (r binder.Binder) {
	if ctx != nil {
		if len(cts) == 0 {
			cts = append(cts, binder.HeaderContentType, binder.HeaderAccept)
		}
		if i := ctx.Value(share.ReqMetaDataKey); i != nil {
			if meta, ok := i.(map[string]string); ok {
				return binder.GetBinder(meta, cts...)
			}
		}
	}
	return binder.Default
}

func Equal(src, tar any) bool {
	t1 := TypeOf(src)
	t2 := TypeOf(tar)
	if t1.Kind() != t2.Kind() {
		return false
	}
	switch t1.Kind() {
	case reflect.String:
		return src.(string) == tar.(string)
	case reflect.Struct:
		return t1.PkgPath()+t1.Name() == t2.PkgPath()+t2.Name()
	case reflect.Slice:
		var ok bool
		var v1, v2 []string
		if v1, ok = src.([]string); !ok {
			return false
		}
		if v2, ok = tar.([]string); !ok {
			return false
		}
		return EqualSliceString(v1, v2)
	default:
		return false
	}
}

// EqualSliceString 包含相同的字符串（可以顺序不同）
func EqualSliceString(v1, v2 []string) bool {
	if len(v1) != len(v2) {
		return false
	}
	m := map[string]int{}
	for _, k := range v1 {
		m[k] += 1
	}
	for _, k := range v2 {
		m[k] -= 1
		if m[k] < 0 {
			return false
		}
	}
	return true
}

func TypeOf(src any) reflect.Type {
	r := reflect.TypeOf(src)
	for r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	return r
}
