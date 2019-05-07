package util

import (
	"bytes"
	"strings"

	"github.com/zdnscloud/g53"
)

type DomainSet struct {
	hashToDomains map[uint32][]*g53.Name
}

func NewDomainSet() *DomainSet {
	return &DomainSet{
		hashToDomains: make(map[uint32][]*g53.Name),
	}
}

func (set *DomainSet) Add(name *g53.Name) {
	if set.Include(name) {
		return
	}

	hash := name.Hash(false)
	names := []*g53.Name{}
	if names_, hit := set.hashToDomains[hash]; hit {
		names = append(names_, name)
	} else {
		names = []*g53.Name{name}
	}
	set.hashToDomains[hash] = names
}

func (set *DomainSet) Include(name *g53.Name) bool {
	hash := name.Hash(false)
	names, hit := set.hashToDomains[hash]
	if hit == false {
		return false
	}

	for _, n := range names {
		if name.Equals(n) {
			return true
		}
	}
	return false
}

const nameSpliter = ", "

func (set *DomainSet) String() string {
	var buf bytes.Buffer
	for _, names := range set.hashToDomains {
		for _, n := range names {
			buf.WriteString(n.String(false))
			buf.WriteString(nameSpliter)
		}
	}
	return strings.TrimRight(buf.String(), nameSpliter)
}
