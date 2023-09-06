package jsonrpc

import (
	"encoding/json"
	"github.com/hwcer/cosgo/values"
	"testing"
)

func TestArgs_Reply(t *testing.T) {
	v := map[string]any{}
	v["id"] = 1
	v["method"] = "test"
	v["params"] = map[string]any{"id": 1001, "num": 10}

	b, _ := json.Marshal(v)

	args, err := NewArgs(b)
	if err != nil {
		t.Logf("%v", err)
	} else {
		t.Logf("%+v", args)
	}

	params := map[string]any{}
	_ = args.Params.Unmarshal(&params)
	t.Logf("%v", params)

	m := values.NewMessage("SUCCESS")

	r := args.Reply(m)
	b2, _ := json.Marshal(r)

	t.Logf("%v", string(b2))
}
