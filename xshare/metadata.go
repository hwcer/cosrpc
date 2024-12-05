package xshare

import (
	"fmt"
)

type Metadata map[string]string

func (this Metadata) Set(k string, v any) {
	this[k] = fmt.Sprintf("%v", v)
}

func (this Metadata) SetContentType(v string) {
	this["Content-Type"] = v
}

func (this Metadata) Json() map[string]string {
	return this
}
