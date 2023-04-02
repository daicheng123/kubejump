package utils

import "strings"

func StringBuild(strs ...string) string {
	var strBuilder = strings.Builder{}
	for _, s := range strs {
		strBuilder.WriteString(s)
	}
	return strBuilder.String()
}
