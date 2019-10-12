package randomdata

import (
	"math/rand"
	"net"
	"time"
)

func SeedUsingNow() {
	rand.Seed(time.Now().UnixNano())
}

func RandUint16() uint16 {
	num := rand.Uint32()
	return uint16(num & 0x0000ffff)
}

func RandV4IP() net.IP {
	num := rand.Uint32()
	bs := make([]byte, 4)
	bs[0] = uint8((num & 0xff000000) >> 24)
	bs[1] = uint8((num & 0x00ff0000) >> 16)
	bs[2] = uint8((num & 0x0000ff00) >> 8)
	bs[3] = uint8(num & 0x000000ff)
	return net.IP(bs[:])
}
