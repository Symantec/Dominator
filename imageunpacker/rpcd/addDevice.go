package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) AddDevice(conn *srpc.Conn) error {
	t.addDeviceLock.Lock()
	defer t.addDeviceLock.Unlock()
	if err := t.unpacker.PrepareForAddDevice(); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	deviceId, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	deviceId = deviceId[:len(deviceId)-1]
	if err := t.unpacker.AddDevice(deviceId); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	_, err = conn.WriteString("\n")
	return err
}
