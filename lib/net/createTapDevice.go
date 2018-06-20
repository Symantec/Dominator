// +build !linux

package net

import (
	"errors"
	"os"
)

func createTapDevice() (*os.File, string, error) {
	return nil, "", errors.New("tap devices not implemented on this OS")
}
