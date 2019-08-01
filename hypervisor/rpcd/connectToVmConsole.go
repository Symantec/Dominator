package rpcd

import (
	"io"

	"github.com/Symantec/Dominator/lib/bufwriter"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ConnectToVmConsole(conn *srpc.Conn) error {
	var request hypervisor.ConnectToVmConsoleRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	console, err := t.manager.ConnectToVmConsole(request.IpAddress,
		conn.GetAuthInformation())
	if console != nil {
		defer console.Close()
	}
	e := conn.Encode(hypervisor.ConnectToVmConsoleResponse{
		Error: errors.ErrorToString(err)})
	if e != nil {
		return e
	}
	if e := conn.Flush(); e != nil {
		return e
	}
	if err != nil {
		return nil
	}
	closed := false
	go func() { // Read from console and write to connection until EOF.
		_, err := io.Copy(bufwriter.NewAutoFlushWriter(conn), console)
		if err != nil && !closed {
			t.logger.Println(err)
		}
		closed = true
	}()
	// Read from connection and write to console.
	_, err = io.Copy(console, conn)
	if err != nil && !closed {
		return err
	}
	closed = true
	return srpc.ErrorCloseClient
}
