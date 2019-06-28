package cas

import (
	"sync"
)

type MemoryStore struct {
	mu    sync.RWMutex
	store map[string]*AuthenticationResponse
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		store: make(map[string]*AuthenticationResponse),
	}
}

func (s *MemoryStore) Read(id string) *AuthenticationResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if t, ok := s.store[id]; ok {
		return t
	} else {
		return nil
	}
}

func (s *MemoryStore) Write(id string, ticket *AuthenticationResponse) error {
	s.mu.Lock()
	s.store[id] = ticket
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	delete(s.store, id)
	s.mu.Unlock()
	return nil
}

func (s *MemoryStore) Clear() error {
	s.mu.Lock()
	s.store = make(map[string]*AuthenticationResponse)
	s.mu.Unlock()
	return nil
}
