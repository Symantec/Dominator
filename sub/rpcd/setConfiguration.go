package rpcd

import (
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) SetConfiguration(request sub.SetConfigurationRequest,
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
