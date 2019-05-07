package domaintree

type RWLocker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

type DumbRWMutex struct {
}

func (l *DumbRWMutex) RLock() {
}

func (l *DumbRWMutex) RUnlock() {
}

func (l *DumbRWMutex) Lock() {
}

func (l *DumbRWMutex) Unlock() {
}
