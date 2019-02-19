package net

import (
	"io/ioutil"
	"path/filepath"
)

func testCarrier(name string) bool {
	filename := filepath.Join(sysClassNet, name, "carrier")
	if data, err := ioutil.ReadFile(filename); err == nil {
		if len(data) > 0 && data[0] == '1' {
			return true
		}
	}
	return false
}
