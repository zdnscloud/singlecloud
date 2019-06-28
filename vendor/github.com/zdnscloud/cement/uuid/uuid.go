package uuid

import (
	"crypto/rand"
	"encoding/hex"
)

func Gen() (string, error) {
	uuid := make([]byte, 16)
	n, err := rand.Read(uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	uuid[8] = 0x80
	uuid[4] = 0x40
	return hex.EncodeToString(uuid), nil
}

func MustGen() string {
	id, err := Gen()
	if err != nil {
		panic("gen uuid failed" + err.Error())
	}
	return id
}
