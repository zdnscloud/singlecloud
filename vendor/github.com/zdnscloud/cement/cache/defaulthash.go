package cache

import (
	"hash/fnv"
)

func HashString(s string) Key {
	h := fnv.New64a()
	h.Write([]byte(s))
	return Key(h.Sum64())
}
