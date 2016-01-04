package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) GetConfiguration(conn *srpc.Conn) error {
	defer conn.Flush()
	var request sub.GetConfigurationRequest
	var response sub.GetConfigurationResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.getConfiguration(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *rpcType) getConfiguration(request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	var response sub.GetConfigurationResponse
	response.ScanSpeedPercent =
		t.scannerConfiguration.FsScanContext.GetContext().SpeedPercent()
	response.NetworkSpeedPercent =
		t.scannerConfiguration.NetworkReaderContext.SpeedPercent()
	response.ScanExclusionList = make([]string,
		len(t.scannerConfiguration.ScanFilter.FilterLines))
	for index, line := range t.scannerConfiguration.ScanFilter.FilterLines {
		response.ScanExclusionList[index] = line
	}
	*reply = response
	return nil
}
