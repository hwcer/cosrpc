package cosrpc

import (
	"reflect"
)

//func ParseError(err error) error {
//	if err == client.ErrXClientNoServer {
//		err = values.NewError(404, err)
//	} else if strings.HasPrefix(err.Error(), "rpcx:") {
//		msg := strings.TrimPrefix(err.Error(), "rpcx:")
//		err = values.NewError(404, strings.Trim(msg, " "))
//	} else {
//		err = values.NewError(0, err)
//	}
//	return err
//}

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
