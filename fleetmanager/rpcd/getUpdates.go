package rpcd

import (
	"errors"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

const flushDelay = time.Millisecond * 10

func (t *srpcType) GetUpdates(conn *srpc.Conn) error {
	var request proto.GetUpdatesRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	closeChannel := conn.GetCloseNotifier()
	updateChannel := t.hypervisorsManager.MakeUpdateChannel(request.Location)
	defer t.hypervisorsManager.CloseUpdateChannel(updateChannel)
	flushTimer := time.NewTimer(flushDelay)
	var numToFlush uint
	maxUpdates := request.MaxUpdates
	for count := uint64(0); maxUpdates < 1 || count < maxUpdates; count++ {
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
			if update.Error != "" {
				return nil
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
	return nil
}
