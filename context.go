package cosrpc

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/logger"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/share"
)

// ICtx 定义上下文接口
// 用于处理 RPC 请求和响应
type ICtx interface {
	Get(key any) any
	SetValue(key, val any)
	Payload() []byte
	Metadata() map[string]string
	ServicePath() string
	ServiceMethod() string
	Write(reply any) error
}

// NewContext 创建并返回一个新的 Context 实例
// 包装传入的 ICtx 接口
func NewContext(ctx ICtx) *Context {
	return &Context{ctx: ctx}
}

// Context 是 cosrpc 上下文的核心结构
// 封装了 ICtx 接口并提供了便捷的方法
type Context struct {
	ctx  ICtx          // 底层的上下文接口
	body values.Values // 请求体的解析结果
}

// Binder 获取绑定器
// 根据元数据和内容类型模式获取对应的绑定器
func (this *Context) Binder(mod ...binder.ContentTypeMod) binder.Binder {
	var t binder.ContentTypeMod
	if len(mod) > 0 {
		t = mod[0]
	} else {
		t = binder.ContentTypeModReq
	}
	return binder.GetContentType(this.Metadata(), t)
}

// Reader 返回一个 io.Reader 来读取包体
func (this *Context) Reader() io.Reader {
	return bytes.NewReader(this.ctx.Payload())
}

// Bytes 返回包体的字节数组
func (this *Context) Bytes() []byte {
	return this.ctx.Payload()
}

// Write 写入响应数据
func (this *Context) Write(data []byte) error {
	return this.ctx.Write(data)
}

// Bind 绑定请求数据到指定的结构体
func (this *Context) Bind(i interface{}) error {
	data := this.ctx.Payload()
	if len(data) == 0 {
		return nil
	}
	bind := this.Binder(binder.ContentTypeModReq)
	return bind.Unmarshal(data, i)
}

// Conn 获取网络连接
func (this *Context) Conn() net.Conn {
	return this.GetValue(server.RemoteConnContextKey).(net.Conn)
}

// Error 创建一个错误消息
func (this *Context) Error(err any) *values.Message {
	return values.Error(err)
}

// Errorf 创建一个带错误码的错误消息
func (this *Context) Errorf(code int32, format any, args ...interface{}) *values.Message {
	return values.Errorf(code, format, args...)
}

// Get 获取请求体中的值
func (this *Context) Get(key string) interface{} {
	v := this.values()
	return v.Get(key)
}

// GetInt 获取请求体中的整数
func (this *Context) GetInt(key string) (val int) {
	v := this.values()
	return v.GetInt(key)
}

// GetInt32 获取请求体中的 int32 类型值
func (this *Context) GetInt32(key string) (val int32) {
	v := this.values()
	return v.GetInt32(key)
}

// GetInt64 获取请求体中的 int64 类型值
func (this *Context) GetInt64(key string) (val int64) {
	v := this.values()
	return v.GetInt64(key)
}

// GetString 获取请求体中的字符串
func (this *Context) GetString(key string) (val string) {
	v := this.values()
	return v.GetString(key)
}

// GetValue 获取上下文中的值
func (this *Context) GetValue(key any) any {
	return this.ctx.Get(key)
}

// SetValue 设置上下文中的值
func (this *Context) SetValue(key, val any) {
	this.ctx.SetValue(key, val)
}

// Metadata 获取元数据
func (this *Context) Metadata() map[string]string {
	return this.ctx.Metadata()
}

// GetMetadata 获取请求元数据
func (this *Context) GetMetadata(key string) (val string) {
	return this.ctx.Metadata()[key]
}

// SetMetadata 设置响应元数据
func (this *Context) SetMetadata(key string, val any) {
	i := this.ctx.Get(share.ResMetaDataKey)
	meta, _ := i.(map[string]string)
	if meta == nil {
		meta = make(map[string]string)
	}
	switch v := val.(type) {
	case string:
		meta[key] = v
	default:
		meta[key] = fmt.Sprintf("%v", val)
	}
	this.ctx.SetValue(share.ResMetaDataKey, meta)
}

// ServicePath 获取服务路径
func (this *Context) ServicePath() string {
	return this.ctx.ServicePath()
}

// ServiceMethod 获取服务方法
func (this *Context) ServiceMethod() string {
	return this.ctx.ServiceMethod()
}

// values 获取解析后的请求体
// 如果未解析，先解析再返回
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
