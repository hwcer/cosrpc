package xshare

import (
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"strconv"
)

const (
	//ServicesMetadataVersion      = "ver"
	ServicesMetadataAverage = "Average"
	//ServicesMetadataServerId     = "sid"

	ServicesMetadataRpcServerId      = "_rpc_srv_id"   //服务器编号
	ServicesMetadataRpcServerAddress = "_rpc_srv_addr" //rpc服务器ID,selector 中固定转发地址
	//ServicesMetadataNetRequestId = "_net_req_id"
)

// NewMetadata 创建新Metadata，参数k1,v1,k2,v2...
func NewMetadata(args ...string) Metadata {
	r := Metadata{}
	var i, j int
	for i = 0; i < len(args)-1; i += 2 {
		j = i + 1
		r[args[i]] = args[j]
	}
	return r
}

type Metadata map[string]string

func (this Metadata) Set(k string, v any) {
	this[k] = fmt.Sprintf("%v", v)
}

func (this Metadata) SetAddress(v string) {
	this[ServicesMetadataRpcServerAddress] = v
}

func (this Metadata) SetServerId(v int32) {
	this[ServicesMetadataRpcServerId] = strconv.Itoa(int(v))
}
func (this Metadata) SetContentType(v string) {
	this[binder.ContentType] = v
}

func (this Metadata) Json() map[string]string {
	return this
}
