package queue

import "container/list"

func newDataQueue() (chan<- interface{}, <-chan interface{}) {
	send := make(chan interface{}, 1)
	receive := make(chan interface{}, 1)
	go manageDataQueue(send, receive)
	return send, receive
}

func manageDataQueue(send <-chan interface{}, receive chan<- interface{}) {
	queue := list.New()
	for {
		if front := queue.Front(); front == nil {
			if send == nil {
				close(receive)
				return
			}
			value, ok := <-send
			if !ok {
				close(receive)
				return
			}
			queue.PushBack(value)
		} else {
			select {
			case receive <- front.Value:
				queue.Remove(front)
			case value, ok := <-send:
				if ok {
					queue.PushBack(value)
				} else {
					send = nil
				}
			}
		}
	}
}
