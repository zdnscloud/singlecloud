package util

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
)

var (
	idLock sync.Mutex
	idRand *rand.Rand
)

func GenMessageId() uint16 {
	idLock.Lock()
	defer idLock.Unlock()

	if idRand == nil {
		// seeding idRand upon the first call to id.
		var seed int64
		var buf [8]byte

		if _, err := crand.Read(buf[:]); err == nil {
			seed = int64(binary.LittleEndian.Uint64(buf[:]))
		} else {
			seed = rand.Int63()
		}

		idRand = rand.New(rand.NewSource(seed))
	}

	// The call to idRand.Uint32 must be within the
	// mutex lock because *rand.Rand is not safe for
	// concurrent use.
	//
	// There is no added performance overhead to calling
	// idRand.Uint32 inside a mutex lock over just
	// calling rand.Uint32 as the global math/rand rng
	// is internally protected by a sync.Mutex.
	return uint16(idRand.Uint32())
}
