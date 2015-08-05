package sub

import (
	"fmt"
)

func (configuration Configuration) String() string {
	return fmt.Sprintf("ScanSpeedPercent: %d", configuration.ScanSpeedPercent)
}

func (configuration GetConfigurationResponse) String() string {
	return Configuration(configuration).String()
}
