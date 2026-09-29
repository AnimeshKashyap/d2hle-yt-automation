package runtime

import "sync"

type ProfileLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewProfileLocks() *ProfileLocks {
	return &ProfileLocks{locks: make(map[string]*sync.Mutex)}
}

func (p *ProfileLocks) Lock(profileID string) func() {
	p.mu.Lock()
	m, ok := p.locks[profileID]
	if !ok {
		m = &sync.Mutex{}
		p.locks[profileID] = m
	}
	p.mu.Unlock()

	m.Lock()
	return func() { m.Unlock() }
}
