package cosrpc

import (
	"github.com/hwcer/cosgo/values"
)

// 错误码定义
const (
	ErrCodeUnknown            = 0   // 未知错误
	ErrCodeInvalidRequest     = 400 // 无效请求
	ErrCodeUnauthorized       = 401 // 未授权
	ErrCodeForbidden          = 403 // 禁止访问
	ErrCodeNotFound           = 404 // 资源不存在
	ErrCodeInternalError      = 500 // 内部错误
	ErrCodeServiceUnavailable = 503 // 服务不可用
)

// Error 将错误转换为 values.Message
func Error(err error) *values.Message {
	if err == nil {
		return nil
	}
	return values.Errorf(ErrCodeUnknown, err)
}

// Errorf 创建一个带错误码的 values.Message
func Errorf(code int32, format string, args ...interface{}) *values.Message {
	return values.Errorf(code, format, args...)
}

// ErrorMsg 创建一个带错误码和错误消息的 values.Message
func ErrorMsg(code int32, message string) *values.Message {
	return values.Errorf(code, message)
}

// ErrorWrap 包装一个错误为 values.Message
func ErrorWrap(code int32, message string, err error) *values.Message {
	if err == nil {
		return nil
	}
	return values.Errorf(code, "%s: %v", message, err)
}
