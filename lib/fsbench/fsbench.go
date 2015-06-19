package fsbench

import (
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"
)

const (
	O_DIRECT    = 00040000
	BUFLEN      = 1024 * 1024
	MAX_TO_READ = 1024 * 1024 * 128
)

func openDirect(name string, flag int, perm os.FileMode) (file *os.File,
	err error) {
	return os.OpenFile(name, flag|O_DIRECT, perm)
}

func getDevnumForFile(name string) (devnum uint64, err error) {
	var stat syscall.Stat_t
	err = syscall.Stat(name, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Dev, nil
}

func getDevnumForDevice(name string) (devnum uint64, err error) {
	var stat syscall.Stat_t
	err = syscall.Stat(name, &stat)
	if err != nil {
		return 0, err
	}
	return stat.Rdev, nil
}

func getDevnodeForFile(name string) (devnode string, err error) {
	var devnum uint64
	devnum, err = getDevnumForFile(name)
	fi_list, err := ioutil.ReadDir("/dev")
	if err != nil {
		return "", err
	}
	for _, fi := range fi_list {
		if (fi.Mode()&os.ModeDevice != 0) &&
			(fi.Mode()&os.ModeCharDevice == 0) {
			var dnum uint64
			devpath := path.Join("/dev", fi.Name())
			dnum, err = getDevnumForDevice(devpath)
			if err != nil {
				return "", err
			}
			if dnum == devnum {
				return devpath, nil
			}
		}
	}
	return "", nil
}

// Compute the maximum read speed (in KiB/s) of a block device, given a file
// within a file-system mounted on the block device.
func GetReadSpeed(name string) (speed uint, err error) {
	var devpath string
	devpath, err = getDevnodeForFile(name)
	if err != nil {
		return 0, err
	}
	var file *os.File
	file, err = openDirect(devpath, os.O_RDONLY, 0)
	if err != nil {
		return 0, err
	}
	tread := 0
	buffer := make([]byte, BUFLEN)
	time_start := time.Now()
	for tread < MAX_TO_READ {
		var nread int
		nread, err = file.Read(buffer)
		if err != nil {
			return 0, err
		}
		tread += nread
	}
	return uint(float64(tread>>10) / time.Since(time_start).Seconds()), nil
}
