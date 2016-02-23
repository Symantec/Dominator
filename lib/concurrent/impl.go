package concurrent

import (
	"runtime"
)

func newState(numConcurrent uint) *State {
	state := new(State)
	if numConcurrent > 0 {
		state.semaphore = make(chan struct{}, numConcurrent)
	} else {
		state.semaphore = make(chan struct{}, runtime.NumCPU())
	}
	state.errorChannel = make(chan error)
	return state
}

func (state *State) goRun(doFunc func() error) error {
	if state.entered {
		panic("GoRun is not re-entrant safe")
	}
	state.entered = true
	defer func() { state.entered = false }()
	for {
		select {
		case err := <-state.errorChannel:
			state.pending--
			if err != nil {
				state.reap()
				return err
			}
		case state.semaphore <- struct{}{}:
			state.pending++
			go func() {
				state.errorChannel <- doFunc()
				<-state.semaphore
			}()
			return nil
		}
	}
}

func (state *State) reap() error {
	close(state.semaphore)
	for ; state.pending > 0; state.pending-- {
		if err := <-state.errorChannel; err != nil {
			return err
		}
	}
	close(state.errorChannel)
	return nil
}
