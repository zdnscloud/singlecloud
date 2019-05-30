package set

import (
	"sort"
)

type StringSet map[string]struct{}

func NewStringSet() StringSet {
	return make(StringSet)
}

func StringSetFromSlice(slice []string) StringSet {
	ss := NewStringSet()
	for _, s := range slice {
		ss.Add(s)
	}
	return ss
}

func (ss StringSet) ToSlice() []string {
	slice := make([]string, 0, len(ss))
	for k, _ := range ss {
		slice = append(slice, k)
	}
	return slice
}

func (ss StringSet) ToSortedSlice() []string {
	slice := ss.ToSlice()
	sort.StringSlice(slice).Sort()
	return slice
}

func (ss StringSet) Equal(other StringSet) bool {
	if len(ss) != len(other) {
		return false
	}

	for s := range ss {
		if _, ok := other[s]; ok == false {
			return false
		}
	}

	return true
}

func (ss StringSet) Member(s string) bool {
	_, ok := ss[s]
	return ok
}

func (ss StringSet) Add(s string) {
	ss[s] = struct{}{}
}

func (ss StringSet) Remove(s string) {
	delete(ss, s)
}

func (ss StringSet) Difference(other StringSet) StringSet {
	res := NewStringSet()
	for s := range ss {
		if _, ok := other[s]; ok == false {
			res.Add(s)
		}
	}
	return res
}

func (ss StringSet) Union(other StringSet) StringSet {
	res := NewStringSet()
	for s := range ss {
		if _, ok := other[s]; ok {
			res.Add(s)
		}
	}
	return res
}
