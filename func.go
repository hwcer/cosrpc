package cosrpc

import (
	"github.com/hwcer/cosgo/message"
	"github.com/smallnest/rpcx/client"
)

func ParseError(err error) error {
	if err == client.ErrXClientNoServer {
		err = message.Errorf(404, err)
	} else {
		err = message.Error(err)
	}
	return err
}
