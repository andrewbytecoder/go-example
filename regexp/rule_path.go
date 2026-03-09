package regexp

import "regexp"

var (
	// 正则表达式，用来匹配以下两种路径
	// 1. 纯静态路径 /etc/prometheus/rules.yml
	// 2. 以通配符结尾的路径 /etc/prometheus/*.yml
	patRulePath = regexp.MustCompile(`^[^*]*(\*[^/]*)?$`)
)
