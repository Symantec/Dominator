package rpcd

import (
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) TraceVmMetadata(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request hypervisor.TraceVmMetadataRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	pathChannel := make(chan string, 1024)
	err := t.manager.RegisterVmMetadataNotifier(request.IpAddress,
		conn.GetAuthInformation(), pathChannel)
	if err == nil {
		defer func() {
			t.manager.UnregisterVmMetadataNotifier(request.IpAddress,
				pathChannel)
		}()
	}
	var response hypervisor.TraceVmMetadataResponse
	response.Error = errors.ErrorToString(err)
	if err := encoder.Encode(response); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	if err != nil {
		return nil
	}
	closeChannel := conn.GetCloseNotifier()
	flushDelay := time.Millisecond * 100
	flushTimer := time.NewTimer(flushDelay)
	for {
		select {
		case path, ok := <-pathChannel:
			if !ok {
				_, err := conn.Write([]byte("VM destroyed\n\n"))
				return err
			}
			if _, err := conn.Write([]byte(path + "\n")); err != nil {
				return err
			}
			flushTimer.Reset(flushDelay)
		case <-flushTimer.C:
			if err := conn.Flush(); err != nil {
				return err
			}
		case err := <-closeChannel:
			if err == nil {
				t.logger.Debugf(0, "metadata trace client disconnected: %s\n",
					conn.RemoteAddr())
				return nil
			}
			t.logger.Println(err)
			return err
		}
	}
}
