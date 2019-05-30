package set

type Set map[interface{}]struct{}

func NewSet() *Set {
	s := make(Set)
	return &s
}

func (s *Set) Add(i interface{}) {
	(*s)[i] = struct{}{}
}

func (s *Set) Contains(i ...interface{}) bool {
	for _, val := range i {
		if _, ok := (*s)[val]; !ok {
			return false
		}
	}
	return true
}

func (s *Set) Clear() {
	*s = *NewSet()
}

func (s *Set) Remove(i interface{}) {
	delete(*s, i)
}
