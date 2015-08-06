package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *Subd) GetConfiguration(request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	var response sub.GetConfigurationResponse
	fs := fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	response.ScanSpeedPercent = fs.Configuration().FsScanContext.SpeedPercent()
	*reply = response
	return nil
}
