package randomdata

import (
	"errors"
	"math/rand"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz"
const (
	letterIdxBits = 5                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var errExceedMaxUniqueCount = errors.New("couldn't generate so many unique strings")

func RandString(n int) string {
	return randString(n, letterBytes)
}

func RandStringWithLetter(n int, letter string) string {
	return randString(n, letter)
}

func randString(n int, letterBytes string) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

const alphaCount = 26

func UniqueRandStrings(strlen, n int) ([]string, error) {
	if permutation(alphaCount, strlen) < n {
		return nil, errExceedMaxUniqueCount
	}

	generatedStrs := make(map[string]struct{})
	uniqueStrs := make([]string, 0, n)
	left := n
	for left > 0 {
		s := RandString(strlen)
		if _, ok := generatedStrs[s]; ok == false {
			generatedStrs[s] = struct{}{}
			left = left - 1
			uniqueStrs = append(uniqueStrs, s)
		}
	}
	return uniqueStrs, nil
}
