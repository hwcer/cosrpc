package jsonrpc

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
)

//{"jsonrpc": "2.0", "method": "subtract", "params": [42, 23], "id": 1}

type Args struct {
	Id     int          `json:"id"`
	Method string       `json:"method"`
	Params values.Bytes `json:"params"`
}

type Reply struct {
	Id      int          `json:"id"`
	Error   *Error       `json:"error,omitempty"`
	Result  values.Bytes `json:"result,omitempty"`
	Jsonrpc string       `json:"jsonrpc"`
}

type Error struct {
	Code    int    `json:"code"`
	Data    string `json:"data,omitempty"`
	Message string `json:"message"`
}

func (this *Args) Reply(v any) *Reply {
	return NewReply(this.Id, values.Parse(v))
}

func (this *Args) Errorf(code int, format any, args ...any) *Reply {
	return NewReply(this.Id, values.Errorf(code, format, args...))
}

func (this *Reply) Bytes(b ...binder.Interface) []byte {
	var bind binder.Interface
	if len(b) > 0 {
		bind = b[0]
	} else {
		bind = binder.Json
	}
	r, err := bind.Marshal(this)
	if err != nil {
		v := values.Error(err)
		this.Error = &Error{Code: v.Code, Message: v.String()}
		r, _ = bind.Marshal(this)
	}
	return r
}
