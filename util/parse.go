package util

import "regexp"

func Parse(s interface{}, m map[string]string) interface{} {
	switch s := s.(type) {
	default:
		return s
	case string:
		return ParseString(s, m)
	case []string:
		return ParseArray(s, m)
	case map[string]interface{}:
		return ParseMap(s, m)
	}
}

func ParseString(s string, m map[string]string) string {
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

func ParseArray(l []string, m map[string]string) []string {
	var results []string
	for _, s := range l {
		results = append(results, ParseString(s, m))
	}
	return results
}

func ParseMap(s map[string]interface{}, m map[string]string) map[string]interface{} {
	results := make(map[string]interface{})

	for k, v := range s {
		results[k] = Parse(v, m)
	}
	return results
}
