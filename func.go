package cosrpc

import (
	"github.com/hwcer/cosgo/message"
	"github.com/smallnest/rpcx/client"
	"strings"
)

func ParseError(err error) error {
	if err == client.ErrXClientNoServer {
		err = message.Errorf(404, err)
	} else if strings.HasPrefix(err.Error(), "rpcx:") {
		msg := strings.TrimPrefix(err.Error(), "rpcx:")
		err = message.Errorf(404, strings.Trim(msg, " "))
	} else {
		err = message.Error(err)
	}
	return err
}
