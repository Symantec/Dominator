package rpcd

import (
	"io"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ConnectToVmSerialPort(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request hypervisor.ConnectToVmSerialPortRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	closeNotifier := make(chan error, 1)
	input, output, err := t.manager.ConnectToVmSerialPort(request.IpAddress,
		conn.GetAuthInformation(), request.PortNumber)
	if input != nil {
		defer close(input)
	}
	e := encoder.Encode(hypervisor.ConnectToVmSerialPortResponse{
		Error: errors.ErrorToString(err)})
	if e != nil {
		return e
	}
	if e := conn.Flush(); e != nil {
		return e
	}
	if err != nil {
		return err
	}
	go func() { // Read from connection and write to input until EOF.
		buffer := make([]byte, 256)
		for {
			if nRead, err := conn.Read(buffer); err != nil {
				if err != io.EOF {
					closeNotifier <- err
				} else {
					closeNotifier <- srpc.ErrorCloseClient
				}
				return
			} else {
				for _, char := range buffer[:nRead] {
					input <- char
				}
			}
		}
	}()
	// Read from output until closure or transmission error.
	for {
		select {
		case data, ok := <-output:
			var buffer []byte
			if !ok {
				buffer = []byte("VM serial port closed\n")
			} else {
				buffer = readData(data, output)
			}
			if _, err := conn.Write(buffer); err != nil {
				return err
			}
			if err := conn.Flush(); err != nil {
				return err
			}
			if !ok {
				return srpc.ErrorCloseClient
			}
		case err := <-closeNotifier:
			return err
		}
	}
}

func readData(firstByte byte, moreBytes <-chan byte) []byte {
	buffer := make([]byte, 1, len(moreBytes)+1)
	buffer[0] = firstByte
	for {
		select {
		case char, ok := <-moreBytes:
			if !ok {
				return buffer
			}
			buffer = append(buffer, char)
		default:
			return buffer
		}
	}
}
