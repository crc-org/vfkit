package util

import "strings"

func TrimQuotes(str string) string {
	if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
		str = strings.Trim(str, `"`)
	}

	return str
}
