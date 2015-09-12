package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *rpcType) GetConfiguration(request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	var response sub.GetConfigurationResponse
	fs := fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	configuration := fs.Configuration()
	response.ScanSpeedPercent =
		configuration.FsScanContext.GetContext().SpeedPercent()
	response.NetworkSpeedPercent =
		configuration.NetworkReaderContext.SpeedPercent()
	response.ScanExclusionList = make([]string,
		len(configuration.ScanFilter.FilterLines))
	for index, line := range configuration.ScanFilter.FilterLines {
		response.ScanExclusionList[index] = line
	}
	*reply = response
	return nil
}
