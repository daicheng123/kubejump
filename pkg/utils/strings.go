package utils

import (
	"fmt"
	"strings"
)

func StringBuild(strs ...string) string {
	var strBuilder = strings.Builder{}
	for _, s := range strs {
		strBuilder.WriteString(s)
	}
	return strBuilder.String()
}

func FilterPrefix(strs []string, s string) (r []string) {
	for _, v := range strs {
		if len(v) >= len(s) {
			if v[:len(s)] == s {
				r = append(r, v)
			}
		}
	}
	return r
}

func LongestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}

	isCommonPrefix := func(length int) bool {
		str0, count := strs[0][:length], len(strs)
		for i := 1; i < count; i++ {
			if strs[i][:length] != str0 {
				return false
			}
		}
		return true
	}

	minLength := len(strs[0])
	for _, s := range strs {
		if len(s) < minLength {
			minLength = len(s)
		}

	}

	low, high := 0, minLength
	for low < high {
		mid := (high-low+1)/2 + low
		if isCommonPrefix(mid) {
			low = mid
		} else {
			high = mid - 1
		}

	}
	return strs[0][:low]
}

func Pretty(strs []string, width int) (s string) {
	longestStr := LongestStr(strs)
	length := len(longestStr) + 4
	lineCount := width / length

	for index, str := range strs {
		if index == 0 {
			s += fmt.Sprintf(fmt.Sprintf("%%-%ds", length), str)
		} else {
			if index%lineCount == 0 {
				s += fmt.Sprintf(fmt.Sprintf("\n%%-%ds", length), str)
			} else {
				s += fmt.Sprintf(fmt.Sprintf("%%-%ds", length), str)
			}
		}
	}

	return s
}

func LongestStr(strs []string) string {
	longestStr := ""
	for _, str := range strs {
		if len(str) >= len(longestStr) {
			longestStr = str
		}
	}

	return longestStr
}
