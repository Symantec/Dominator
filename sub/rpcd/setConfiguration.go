package rpcd

import (
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) SetConfiguration(conn *srpc.Conn,
	request sub.SetConfigurationRequest,
	reply *sub.SetConfigurationResponse) error {
	t.scannerConfiguration.FsScanContext.GetContext().SetSpeedPercent(
		request.ScanSpeedPercent)
	t.scannerConfiguration.NetworkReaderContext.SetSpeedPercent(
		request.NetworkSpeedPercent)
	newFilter, err := filter.New(request.ScanExclusionList)
	if err != nil {
		return err
	}
	t.scannerConfiguration.ScanFilter = newFilter
	t.logger.Printf("SetConfiguration()\n")
	return nil
}
