package rpcd

import (
	"fmt"
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ProbeVmPort(conn *srpc.Conn,
	request hypervisor.ProbeVmPortRequest,
	reply *hypervisor.ProbeVmPortResponse) error {
	if _, err := t.manager.GetVmInfo(request.IpAddress); err != nil {
		*reply = hypervisor.ProbeVmPortResponse{
			Error: errors.ErrorToString(err)}
		return nil
	}
	ok, err := probeVmPort(
		fmt.Sprintf("%s:%d", request.IpAddress, request.PortNumber),
		request.Timeout)
	*reply = hypervisor.ProbeVmPortResponse{ok, errors.ErrorToString(err)}
	return nil
}

func probeVmPort(addr string, timeout time.Duration) (bool, error) {
	// TODO(rgooch): This should be done inside the metadata server namespace
	//               and it should be limited to VM owners.
	intervalDuration := time.Millisecond * 250
	intervalTimer := time.NewTimer(intervalDuration)
	okChannel := make(chan struct{}, 1)
	timeoutTimer := time.NewTimer(timeout)
	for {
		select {
		case <-intervalTimer.C:
			go probeOnce(addr, okChannel)
			intervalTimer.Reset(intervalDuration)
		case <-okChannel:
			return true, nil
		case <-timeoutTimer.C:
			return false, nil
		}
	}
}

func probeOnce(addr string, okChannel chan<- struct{}) {
	if conn, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		conn.Close()
		select {
		case okChannel <- struct{}{}:
		default:
		}
	}
}
