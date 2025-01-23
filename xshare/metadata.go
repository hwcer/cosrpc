package xshare

import (
	"fmt"
	"strconv"
)

// NewMetadata 创建新Metadata，参数k1,v1,k2,v2...
func NewMetadata(args ...string) Metadata {
	r := Metadata{}
	var i, j int
	for i = 0; i < len(args)-1; i += 2 {
		j = i + 1
		r[args[i]] = args[j]
	}
	return r
}

type Metadata map[string]string

func (meta Metadata) Set(k string, v any) {
	switch i := v.(type) {
	case string:
		meta[k] = i
	default:
		meta[k] = fmt.Sprintf("%v", v)
	}
}

func (meta Metadata) SetAddress(v string) {
	meta[ServiceSelectorServerAddress] = v
}

func (meta Metadata) SetServerId(v int32) {
	meta[ServiceSelectorServerId] = strconv.Itoa(int(v))
}
func (meta Metadata) SetContentType(v string) {
	meta["Content-Type"] = v
}

//func (meta Metadata) Json() map[string]string {
//	return meta
//}

func (meta Metadata) Get(k string) string {
	return meta[k]
}

func (meta Metadata) GetInt(k string) int {
	return int(meta.GetInt64(k))
}

func (meta Metadata) GetInt32(k string) int32 {
	return int32(meta.GetInt64(k))
}

func (meta Metadata) GetInt64(k string) int64 {
	s := meta[k]
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func (meta Metadata) GetFloat32(k string) float32 {
	return float32(meta.GetFloat64(k))
}

func (meta Metadata) GetFloat64(k string) (r float64) {
	s := meta[k]
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseFloat(s, 64)
	return i
}
func (meta Metadata) GetString(k string) (r string) {
	return meta[k]
}
