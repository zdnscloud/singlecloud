package util

import (
	"strconv"
	"strings"
)

func HexStrToBytes(s string) (result []uint8, err error) {
	for i := 0; i+1 < len(s); i += 2 {
		d, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err != nil {
			break
		}
		result = append(result, uint8(d))
	}
	return
}

func BytesToElixirStr(bytes []uint8) string {
	str := "<<"
	for _, b := range bytes {
		str += strconv.Itoa(int(b)) + ","
	}
	return str + ">>"
}

func StringSliceCompare(strs1 []string, strs2 []string, caseSensitive bool) int {
	len1 := len(strs1)
	len2 := len(strs2)
	minLen := len1
	if len2 < len1 {
		minLen = len2
	}

	for i := 0; i < minLen; i++ {
		s1 := strs1[i]
		s2 := strs2[i]
		if caseSensitive == false {
			s1 = strings.ToLower(s1)
			s2 = strings.ToLower(s2)
		}

		if order := strings.Compare(s1, s2); order != 0 {
			return order
		}
	}

	if len1 == len2 {
		return 0
	} else if len1 > len2 {
		return 1
	} else {
		return -1
	}
}
