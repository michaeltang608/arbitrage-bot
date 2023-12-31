package util

import "strings"

func SliceContains(slice []string, ele string, caseSensitive bool) (index int) {
	for idx, e := range slice {
		if e == ele {
			return idx
		} else {
			if !caseSensitive && strings.ToUpper(e) == strings.ToUpper(ele) {
				return idx
			}
		}
	}
	return -1
}
