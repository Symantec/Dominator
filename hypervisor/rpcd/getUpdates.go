package rpcd

import (
	"errors"
	"time"

	"github.com/Symantec/Dominator/lib/srpc"
)

const flushDelay = time.Millisecond * 10

func (t *srpcType) GetUpdates(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	closeChannel := conn.GetCloseNotifier()
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
			if err := encoder.Encode(update); err != nil {
				t.logger.Printf("error sending update: %s\n", err)
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
