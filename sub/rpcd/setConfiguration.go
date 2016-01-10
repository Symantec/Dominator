package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) SetConfiguration(conn *srpc.Conn) error {
	defer conn.Flush()
	var request sub.SetConfigurationRequest
	var response sub.SetConfigurationResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.setConfiguration(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *rpcType) setConfiguration(request sub.SetConfigurationRequest,
	reply *sub.SetConfigurationResponse) error {
	var response sub.SetConfigurationResponse
	t.scannerConfiguration.FsScanContext.GetContext().SetSpeedPercent(
		request.ScanSpeedPercent)
	t.scannerConfiguration.NetworkReaderContext.SetSpeedPercent(
		request.NetworkSpeedPercent)
	newFilter, err := filter.NewFilter(request.ScanExclusionList)
	if err != nil {
		return err
	}
	t.scannerConfiguration.ScanFilter = newFilter
	response.Success = true
	*reply = response
	return nil
}
