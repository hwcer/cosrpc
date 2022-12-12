package cosrpc

import (
	"github.com/hwcer/cosgo/values"
	"github.com/smallnest/rpcx/client"
	"strings"
)

func ParseError(err error) error {
	if err == client.ErrXClientNoServer {
		err = values.NewError(404, err)
	} else if strings.HasPrefix(err.Error(), "rpcx:") {
		msg := strings.TrimPrefix(err.Error(), "rpcx:")
		err = values.NewError(404, strings.Trim(msg, " "))
	} else {
		err = values.NewError(0, err)
	}
	return err
}
