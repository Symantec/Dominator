package sub

import (
	"fmt"
)

func (configuration Configuration) String() string {
	retval := fmt.Sprintf("CpuPercent: %d\nNetworkSpeedPercent: %d\nScanSpeedPercent: %d",
		configuration.CpuPercent, configuration.NetworkSpeedPercent,
		configuration.ScanSpeedPercent)
	if len(configuration.ScanExclusionList) > 0 {
		retval += "\n" + "ScanExclusionList:"
		for _, exclusion := range configuration.ScanExclusionList {
			retval += "\n  " + exclusion
		}
	}
	return retval
}

func (configuration GetConfigurationResponse) String() string {
	return Configuration(configuration).String()
}
