package util

import "regexp"

func Parse(s string, m map[string]string) string {
	r, _ := regexp.Compile("{{([^{}]+)}}")
	return r.ReplaceAllStringFunc(s, func(s string) string {
		k := s[2 : len(s)-2]
		v, ok := m[k]
		if !ok {
			return s
		}
		return v
	})
}
