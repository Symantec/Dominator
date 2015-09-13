package rpcd

import (
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) GetConfiguration(request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	var response sub.GetConfigurationResponse
	response.ScanSpeedPercent =
		scannerConfiguration.FsScanContext.GetContext().SpeedPercent()
	response.NetworkSpeedPercent =
		scannerConfiguration.NetworkReaderContext.SpeedPercent()
	response.ScanExclusionList = make([]string,
		len(scannerConfiguration.ScanFilter.FilterLines))
	for index, line := range scannerConfiguration.ScanFilter.FilterLines {
		response.ScanExclusionList[index] = line
	}
	*reply = response
	return nil
}
