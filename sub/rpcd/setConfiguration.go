package rpcd

import (
	"errors"
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
	configuration.FsScanContext.SetSpeedPercent(request.ScanSpeedPercent)
	err := configuration.SetExclusionList(request.ScanExclusionList)
	if err != nil {
		return err
	}
	response.Success = true
	*reply = response
	return nil
}
