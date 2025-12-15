// Package file contains file utilities
package file

import "strings"

func ExtractSuffix(s string) (suffix string, idx int) {
	idx = strings.LastIndex(s, ".")
	if idx == -1 {
		return s, idx
	}
	return s[idx:], idx
}
