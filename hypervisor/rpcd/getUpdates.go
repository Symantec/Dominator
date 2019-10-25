package rpcd

import (
	"errors"
	"io"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

const flushDelay = time.Millisecond * 10

func (t *srpcType) GetUpdates(conn *srpc.Conn) error {
	closeChannel, responseChannel := t.getUpdatesReader(conn)
	updateChannel := t.manager.MakeUpdateChannel()
	defer t.manager.CloseUpdateChannel(updateChannel)
	flushTimer := time.NewTimer(flushDelay)
	var numToFlush uint
	for {
		select {
		case update, ok := <-updateChannel:
			if !ok {
				err := errors.New("receiver not keeping up with updates")
				t.logger.Printf("error sending update: %s\n", err)
				return err
			}
			if err := conn.Encode(update); err != nil {
				t.logger.Printf("error sending update: %s\n", err)
				return err
			}
			numToFlush++
			flushTimer.Reset(flushDelay)
		case update, ok := <-responseChannel:
			if !ok {
				err := errors.New("receiver not keeping up with reponses")
				t.logger.Printf("error sending response: %s\n", err)
				return err
			}
			if err := conn.Encode(update); err != nil {
				t.logger.Printf("error sending response: %s\n", err)
				return err
			}
			numToFlush++
			flushTimer.Reset(flushDelay)
		case <-flushTimer.C:
			if numToFlush > 1 {
				t.logger.Debugf(0, "flushing %d events\n", numToFlush)
			}
			numToFlush = 0
			if err := conn.Flush(); err != nil {
				t.logger.Printf("error flushing update(s): %s\n", err)
				return err
			}
		case err := <-closeChannel:
			if err == nil {
				t.logger.Debugf(0, "update client disconnected: %s\n",
					conn.RemoteAddr())
				return nil
			}
			t.logger.Println(err)
			return err
		}
	}
}

func (t *srpcType) getUpdatesReader(decoder srpc.Decoder) (
	<-chan error, <-chan proto.Update) {
	closeChannel := make(chan error)
	responseChannel := make(chan proto.Update, 16)
	go func() {
		for {
			var request proto.GetUpdateRequest
			if err := decoder.Decode(&request); err != nil {
				if err == io.EOF {
					err = nil
				}
				closeChannel <- err
				return
			}
			update := proto.Update{HealthStatus: t.manager.GetHealthStatus()}
			select {
			case responseChannel <- update:
			default:
				close(responseChannel)
				return
			}
		}
	}()
	return closeChannel, responseChannel
}
