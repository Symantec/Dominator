package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) GetConfiguration(conn *srpc.Conn) {
	defer conn.Flush()
	var request sub.GetConfigurationRequest
	var response sub.GetConfigurationResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	if err := t.getConfiguration(request, &response); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	conn.WriteString("\n")
	encoder := gob.NewEncoder(conn)
	encoder.Encode(response)
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
