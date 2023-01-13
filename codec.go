package cosrpc

import (
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/share"
)

const SerializeType protocol.SerializeType = 200

func init() {
	share.RegisterCodec(SerializeType, &Codec{})
}

// Codec uses raw slice pf bytes and don't encode/decode.
type Codec struct{}

// Encode returns raw slice of bytes.
func (c Codec) Encode(i interface{}) ([]byte, error) {
	return nil, nil
}

// Decode returns raw slice of bytes.
func (c Codec) Decode(data []byte, i interface{}) error {
	return nil
}
