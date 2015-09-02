package util

import (
	"metronome/config"
	"regexp"

	"github.com/imdario/mergo"
)

//	Parse {{XXXX}} to value by appropriate method depending on type of src
func Parse(src interface{}, vars map[string]string) interface{} {
	switch src := src.(type) {
	default:
		return src
	case string:
		return ParseString(src, vars)
	case []string:
		return ParseArray(src, vars)
	case map[string]interface{}:
		return ParseMap(src, vars)
	}
}

//	Parse {{XXXX}} in string to value
func ParseString(src string, vars map[string]string) string {
	mergo.Merge(&vars, map[string]string(config.UserVariables))

	r := regexp.MustCompile("{{([^{}]+)}}")
	for prev := ""; prev != src; {
		prev = src
		src = r.ReplaceAllStringFunc(src, func(s string) string {
			r2 := regexp.MustCompile(`{{config\.([^{}]+)}}`)
			matches := r2.FindStringSubmatch(s)
			if len(matches) == 2 {
				return config.GetValue(matches[1])
			} else {
				k := s[2 : len(s)-2]
				v, ok := vars[k]
				if !ok {
					return s
				}
				return v
			}
		})
	}

	return src
}

//	Parse {{XXXX}} in array of string to value
func ParseArray(src []string, vars map[string]string) []string {
	var results []string
	for _, e := range src {
		results = append(results, ParseString(e, vars))
	}
	return results
}

//	Parse {{XXXX}} in map to value
func ParseMap(src map[string]interface{}, vars map[string]string) map[string]interface{} {
	results := make(map[string]interface{})

	for k, v := range src {
		results[k] = Parse(v, vars)
	}
	return results
}
