package xshare

var Selector = selector{} //预设选择器

type selector map[string]any

func (s selector) Set(servicePath string, selectorType any) {
	s[servicePath] = selectorType
}

func (s selector) Get(servicePath string) any {
	return s[servicePath]
}
