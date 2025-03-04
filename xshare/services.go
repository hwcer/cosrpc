package xshare

const (
	SelectorTypeLocal     = "local"     //本地程序内访问
	SelectorTypeProcess   = "process"   //进程内访问
	SelectorTypeDiscovery = "discovery" //服务发现
)

var Service = service{}

type service map[string]string

func (s service) Get(servicePath string) string {
	return s[servicePath]
}
func (s service) Set(servicePath string, value string) {
	s[servicePath] = value
}

// Discovery 是否有必要使用服务器发现
func (s service) Discovery() bool {
	for _, v := range s {
		if v == SelectorTypeDiscovery {
			return true
		}
	}
	return false
}
