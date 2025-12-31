package generator

import "strings"

// toLowerFirst 将字符串首字母转为小写
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(string(s[0])) + s[1:]
}
