package xshare

var Metadata = metadata{}

type metadata map[string]string

func (meta metadata) Set(servicePath string, data string) {
	meta[servicePath] = data
}

func (meta metadata) Get(servicePath string) string {
	return meta[servicePath]
}
