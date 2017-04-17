package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) GetConfiguration(conn *srpc.Conn,
	request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	var response sub.GetConfigurationResponse
	response = sub.GetConfigurationResponse(t.getConfiguration())
	*reply = response
	return nil
}

func (t *rpcType) getConfiguration() sub.Configuration {
	var configuration sub.Configuration
	configuration.CpuPercent =
		t.scannerConfiguration.CpuLimiter.CpuPercent()
	configuration.NetworkSpeedPercent =
		t.scannerConfiguration.NetworkReaderContext.SpeedPercent()
	configuration.ScanSpeedPercent =
		t.scannerConfiguration.FsScanContext.GetContext().SpeedPercent()
	configuration.ScanExclusionList =
		t.scannerConfiguration.ScanFilter.FilterLines
	return configuration
}
