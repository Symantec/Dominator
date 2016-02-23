package concurrent

type State struct {
	semaphore    chan struct{}
	errorChannel chan error
	pending      uint64
}

func NewState(numConcurrent uint) *State {
	return newState(numConcurrent)
}

func (state *State) GoRun(doFunc func() error) error {
	return state.goRun(doFunc)
}

func (state *State) Reap() error {
	return state.reap()
}
