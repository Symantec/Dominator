package concurrent

import "sync"

type incrementer struct {
	state              *State
	mutex              sync.Mutex // Protect the following fields.
	completionCount    uint
	currentConcurrency uint
}

func newStateWithLinearConcurrencyIncrease(initialNumConcurrent uint,
	maximumNumConcurrent uint) *State {
	if initialNumConcurrent < 1 {
		panic("initialNumConcurrent must not be zero")
	}
	p := &incrementer{currentConcurrency: initialNumConcurrent}
	state := newState(maximumNumConcurrent, p)
	p.state = state
	if initialNumConcurrent > uint(cap(state.semaphore)) {
		panic("initialNumConcurrent must not exceed concurrency")
	}
	numToBlock := uint(cap(state.semaphore)) - initialNumConcurrent
	for count := uint(0); count < numToBlock; count++ {
		state.semaphore <- struct{}{}
	}
	return state
}

// This is called from a goroutine.
func (p *incrementer) put() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.currentConcurrency >= uint(cap(p.state.semaphore)) {
		return
	}
	p.completionCount++
	if p.completionCount < p.currentConcurrency {
		return
	}
	p.completionCount = 0
	p.currentConcurrency++
	select {
	case <-p.state.semaphore:
	default:
		panic("no concurrency limits to remove")
	}
}
