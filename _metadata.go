package cosrpc

import (
	"github.com/hwcer/cosgo/binder"
	"strconv"
)

const (
	MetadataUid          = "uid"
	MetadataGuid         = "guid"
	MetadataCookie       = "Cookie"          //修改网关
	MetadataGateway      = "Gateway"         //网关地址
	MetadataMessagePath  = "Message"         //长连接message id
	MetadataRpcAddress   = "_rpc_address"    //rpc服务器ID,selector 中固定转发地址
	MetadataRpcServerId  = "_rpc_server_id"  //服务器编号
	MetadataNetRequestId = "_net_request_id" //客户端请求ID
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

func (this Metadata) SetUid(v string) {
	this[MetadataUid] = v
}
func (this Metadata) SetAddress(v string) {
	this[MetadataRpcAddress] = v
}

func (this Metadata) SetServerId(v int32) {
	this[MetadataRpcServerId] = strconv.Itoa(int(v))
}
func (this Metadata) SetContentType(v string) {
	this[binder.ContentType] = v
}
