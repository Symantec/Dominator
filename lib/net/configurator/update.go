package configurator

import (
	"github.com/Symantec/Dominator/lib/log"
)

func (netconf *NetworkConfig) update(rootDir string,
	logger log.DebugLogger) (bool, error) {
	updated := false
	if u, err := netconf.updateDebian(rootDir); err != nil {
		return updated, err
	} else if u {
		logger.Printf("updated network interfaces configuration")
		updated = true
	}
	if u, err := updateResolvConf(rootDir, netconf.DefaultSubnet); err != nil {
		return updated, err
	} else if u {
		logger.Printf("updated DNS resolver configuration")
		updated = true
	}
	return updated, nil
}
