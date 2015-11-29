package rpcd

import (
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) GetConfiguration(request sub.GetConfigurationRequest,
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
