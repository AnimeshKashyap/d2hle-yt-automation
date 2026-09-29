package runtime

import "sync"

type SlotPool struct {
	capacity int
	active   int
	mu       sync.Mutex
}

func NewSlotPool(capacity int) *SlotPool {
	return &SlotPool{capacity: capacity}
}

func (p *SlotPool) TryAcquire() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active >= p.capacity {
		return false
	}
	p.active++
	return true
}

func (p *SlotPool) Release() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.active > 0 {
		p.active--
	}
}

func (p *SlotPool) Available() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active < p.capacity
}

func (p *SlotPool) ActiveCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active
}
