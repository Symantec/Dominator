/*
	Package concurrent provides a simple way to run functions concurrently and
	then reap the results.

	Package concurrent allows cuncurrent running of provided functions, by
	default limiting the parallelism to the number of CPUs. The functions return
	an error value and these may be reaped at the end.
*/
package concurrent

// State maintains state needed to manage running functions concurrently.
type State struct {
	entered      bool
	semaphore    chan struct{}
	errorChannel chan error
	pending      uint64
}

// NewState returns a new State.
func NewState(numConcurrent uint) *State {
	return newState(numConcurrent)
}

// GoRun will run the provided function in a goroutine. If the function returns
// a non-nil error, this will be returned in a future call to GoRun or by
// Reap.
func (state *State) GoRun(doFunc func() error) error {
	return state.goRun(doFunc)
}

// Reap returns the first error encountered by the functions and waits for
// remaining functions to complete. The State can no longer be used after Reap.
func (state *State) Reap() error {
	return state.reap()
}
