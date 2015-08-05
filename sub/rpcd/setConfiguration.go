package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/proto/sub"
)

func (t *Subd) SetConfiguration(request sub.SetConfigurationRequest,
	reply *sub.SetConfigurationResponse) error {
	var response sub.SetConfigurationResponse
	fs := fileSystemHistory.FileSystem()
	if fs == nil {
		return errors.New("No file-system history yet")
	}
	fs.FsRateContext().SetSpeedPercent(request.ScanSpeedPercent)
	response.Success = true
	*reply = response
	return nil
}
