package cosrpc

import (
	"github.com/hwcer/cosgo/binding"
	"github.com/hwcer/cosgo/message"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"
)

const ContextMimeTypeName = "_cosrpc_context_mime_type"

//var pool sync.Pool
//
//func init() {
//	pool = sync.Pool{}
//	pool.New = func() interface{} {
//		c := &Context{}
//		return c
//	}
//}

type Context struct {
	*server.Context
	body values.Values
}

//func (this *Context) Reset(s *server.Context) error {
//	this.Context = s
//	return nil
//}
//
//func (this *Context) Release() {
//	this.body = nil
//	this.Context = nil
//}

func (this *Context) SetMimeType(t string) {
	this.Context.SetValue(ContextMimeTypeName, t)
}
func (this *Context) GetMimeType() (r string) {
	v := this.Context.Get(ContextMimeTypeName)
	r, _ = v.(string)
	return
}
func (this *Context) Bind(i interface{}) error {
	data := this.Context.Payload()
	if len(data) == 0 {
		return nil
	}
	mimeType := this.GetMimeType()
	if mimeType != "" {
		return binding.Unmarshal(data, i, mimeType)
	} else {
		return binding.Unmarshal(data, i, binding.MIMEJSON)
	}
}

func (this *Context) Error(err interface{}) *message.Message {
	return message.Error(err)
}

func (this *Context) Errorf(code int, format string, args ...interface{}) *message.Message {
	return message.Errorf(code, format, args...)
}

func (this *Context) Get(key string) interface{} {
	v := this.values()
	return v.Get(key)
}

func (this *Context) GetInt(key string) (val int) {
	v := this.values()
	return v.GetInt(key)
}

func (this *Context) GetInt32(key string) (val int32) {
	v := this.values()
	return v.GetInt32(key)
}

func (this *Context) GetInt64(key string) (val int64) {
	v := this.values()
	return v.GetInt64(key)
}
func (this *Context) GetString(key string) (val string) {
	v := this.values()
	return v.GetString(key)
}

// GetMetadata GET REQ Metadata
func (this *Context) GetMetadata(key string) (val string) {
	return this.Context.Metadata()[key]
}

// SetMetadata SET RES Metadata
func (this *Context) SetMetadata(key, val string) {
	i := this.Context.Get(share.ResMetaDataKey)
	meta, _ := i.(map[string]string)
	if meta == nil {
		meta = make(map[string]string)
	}
	meta[key] = val
	this.Context.SetValue(share.ResMetaDataKey, meta)
}

func (this *Context) values() values.Values {
	if this.body == nil {
		this.body = make(values.Values)
		err := this.Bind(&this.body)
		if err != nil {
			logger.Debug("Context values Unmarshal error:%v", err)
		}
	}
	return this.body
}
