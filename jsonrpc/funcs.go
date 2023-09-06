package jsonrpc

import (
	"encoding/json"
	"github.com/hwcer/cosgo/values"
)

func NewArgs(v []byte) (r *Args, err error) {
	r = &Args{}
	err = json.Unmarshal(v, r)
	return
}

func NewReply(id int, v *values.Message) *Reply {
	r := &Reply{Id: id}
	if v.Code != 0 {
		r.Error = &Error{Code: v.Code}
		if err := v.Unmarshal(&r.Error.Message); err != nil {
			r.Error.Message = err.Error()
		}
	} else {
		r.Result = v.Data
	}
	r.Jsonrpc = "2.0"
	return r
}

func NewError(code int, err any, args ...any) *Reply {
	v := values.Errorf(code, err, args)
	return NewReply(0, v)
}
