package net

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Symantec/Dominator/lib/fsutil"
)

const procNetVlan = "/proc/net/vlan"

func getBridgeVlanId(bridgeIf string) (int, error) {
	dirname := filepath.Join(sysClassNet, bridgeIf, "brif")
	ifNames, err := fsutil.ReadDirnames(dirname, false)
	if err != nil {
		return 0, err
	}
	retval := -1
	for _, ifName := range ifNames {
		if strings.HasPrefix(ifName, "tap") {
			continue
		}
		if strings.HasSuffix(ifName, "-ll") {
			continue
		}
		_, err := os.Stat(filepath.Join(sysClassNet, ifName, "tun_flags"))
		if err == nil {
			continue
		}
		vlanId, err := readVlanId(filepath.Join(procNetVlan, ifName))
		if err != nil {
			return 0, err
		}
		if vlanId >= 0 {
			return int(vlanId), nil
		}
		retval = 0
	}
	return retval, nil
}

func readVlanId(filename string) (uint, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer file.Close()
	var dummy string
	var vlanId uint
	_, err = fmt.Fscanf(file, "%s VID: %d %s", &dummy, &vlanId, &dummy)
	if err != nil {
		return 0, err
	}
	return vlanId, nil
}
