package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) SetConfiguration(request sub.SetConfigurationRequest,
	reply *sub.SetConfigurationResponse) error {
	var response sub.SetConfigurationResponse
	fs := fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	configuration := fs.Configuration()
	configuration.FsScanContext.GetContext().SetSpeedPercent(
		request.ScanSpeedPercent)
	configuration.NetworkReaderContext.SetSpeedPercent(
		request.NetworkSpeedPercent)
	newFilter, err := filter.NewFilter(request.ScanExclusionList)
	if err != nil {
		return err
	}
	configuration.Filter = newFilter
	response.Success = true
	*reply = response
	return nil
}
