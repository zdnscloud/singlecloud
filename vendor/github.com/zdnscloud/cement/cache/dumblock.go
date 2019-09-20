package cache

type DumbLock struct {
}

func (l *DumbLock) Lock() {
}

func (l *DumbLock) Unlock() {
}
