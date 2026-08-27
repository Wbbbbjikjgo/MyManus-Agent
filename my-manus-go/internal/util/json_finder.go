package util

import "encoding/json"

// FindFirstJson 返回字符串中第一个合法的 JSON 对象或数组（对应 Java JsonFinder.findFirstJson）。
func FindFirstJson(src string) (string, bool) {
	for i := 0; i < len(src); i++ {
		var end byte
		switch src[i] {
		case '{':
			end = '}'
		case '[':
			end = ']'
		default:
			continue
		}
		for j := i + 1; j < len(src); j++ {
			if src[j] == end {
				candidate := src[i : j+1]
				if isValidJson(candidate) {
					return candidate, true
				}
			}
		}
	}
	return "", false
}

func isValidJson(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}
