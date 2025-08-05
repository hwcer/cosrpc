package cosrpc

import (
	"context"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/smallnest/rpcx/share"
	"reflect"
	"unsafe"
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

func Unmarshal(src any, tar any) error {
	// 检查 tar 必须是指针
	tarVal := reflect.ValueOf(tar)
	if tarVal.Kind() != reflect.Ptr || tarVal.IsNil() {
		return fmt.Errorf("tar must be a non-nil pointer")
	}

	srcVal := reflect.ValueOf(src)
	tarElem := tarVal.Elem()

	// 获取 tar 的指针地址
	tarPtr := unsafe.Pointer(tarVal.Pointer())

	// 情况1：src 是指针
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return fmt.Errorf("src pointer is nil")
		}

		// 检查类型是否匹配
		if srcVal.Elem().Type() != tarElem.Type() {
			return fmt.Errorf("src and tar must have the same underlying type")
		}

		// 直接让 tar 指向 src 指向的对象
		*(*unsafe.Pointer)(tarPtr) = unsafe.Pointer(srcVal.Pointer())
		return nil
	}

	// 情况2：src 不是指针
	if srcVal.Type() != tarElem.Type() {
		return fmt.Errorf("src and tar must have the same underlying type")
	}

	// 创建 src 的副本（分配在堆上）
	copySrc := reflect.New(srcVal.Type()).Elem()
	copySrc.Set(srcVal)

	// 让 tar 指向这个副本
	*(*unsafe.Pointer)(tarPtr) = unsafe.Pointer(copySrc.Addr().Pointer())

	return nil
}
