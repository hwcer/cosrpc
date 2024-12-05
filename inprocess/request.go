package inprocess

import "github.com/smallnest/rpcx/protocol"

type Request struct {
	st            protocol.SerializeType
	ServicePath   string
	ServiceMethod string
	Payload       []byte
}

// SerializeType returns serialization type of payload.
func (r *Request) SerializeType() protocol.SerializeType {
	return r.st
}

// SetSerializeType sets the serialization type.
func (r *Request) SetSerializeType(st protocol.SerializeType) {
	r.st = st
}
