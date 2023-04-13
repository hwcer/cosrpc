package cosrpc

import (
	"bytes"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"
	"io"
)

type Context struct {
	*server.Context
	body   values.Values
	Binder binder.Interface
}

// Reader 返回一个io.Reader来读取包体
func (this *Context) Reader() io.Reader {
	return bytes.NewReader(this.Context.Payload())
}

func (this *Context) Bind(i interface{}) error {
	data := this.Context.Payload()
	if len(data) == 0 {
		return nil
	}
	var bind binder.Interface
	if t := this.GetMetadata(binder.ContentType); t != "" {
		bind = binder.New(t)
	}
	if bind == nil {
		bind = this.Binder
	}
	return bind.Unmarshal(data, i)
}

func (this *Context) Error(err any) *values.Message {
	return values.Errorf(0, err)
}

func (this *Context) Errorf(code int, format any, args ...interface{}) *values.Message {
	return values.Errorf(code, format, args...)
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
