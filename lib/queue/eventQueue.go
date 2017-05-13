package queue

func newEventQueue() (chan<- struct{}, <-chan struct{}) {
	send := make(chan struct{}, 1)
	receive := make(chan struct{}, 1)
	go manageEventQueue(send, receive)
	return send, receive
}

func manageEventQueue(send <-chan struct{}, receive chan<- struct{}) {
	numInQueue := 0
	for {
		if numInQueue < 1 {
			if send == nil {
				close(receive)
				return
			}
			_, ok := <-send
			if !ok {
				close(receive)
				return
			}
			numInQueue++
		} else {
			select {
			case receive <- struct{}{}:
				numInQueue--
			case _, ok := <-send:
				if ok {
					numInQueue++
				} else {
					send = nil
				}
			}
		}
	}
}
