package inprocess

import "github.com/smallnest/rpcx/protocol"

type Request struct {
	st            protocol.SerializeType
	ServicePath   string
	ServiceMethod string
	Payload       []byte // 序列化后的二进制数据（XCall 路径传入，或由 Args 懒序列化生成）
	Args          any    // 原始参数（inprocess 直传，避免不必要的序列化开销）
}

// SerializeType returns serialization type of payload.
func (r *Request) SerializeType() protocol.SerializeType {
	return r.st
}

// SetSerializeType sets the serialization type.
func (r *Request) SetSerializeType(st protocol.SerializeType) {
	r.st = st
}
