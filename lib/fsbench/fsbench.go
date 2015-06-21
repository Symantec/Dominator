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
func GetReadSpeed(name string) (speed uint, blocksize uint, err error) {
	speed = 0
	blocksize = 0
	var devpath string
	devpath, err = getDevnodeForFile(name)
	if err != nil {
		return
	}
	var file *os.File
	file, err = openDirect(devpath, os.O_RDONLY, 0)
	if err != nil {
		return
	}
	var tread uint = 0
	buffer := make([]byte, BUFLEN)
	var rusage_start, rusage_stop syscall.Rusage
	err = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage_start)
	if err != nil {
		return
	}
	time_start := time.Now()
	for tread < MAX_TO_READ {
		var nread int
		nread, err = file.Read(buffer)
		if err != nil {
			return
		}
		tread += uint(nread)
	}
	speed = uint(float64(tread>>10) / time.Since(time_start).Seconds())
	err = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage_stop)
	if err != nil {
		return
	}
	if rusage_stop.Inblock > rusage_start.Inblock {
		blocksize = uint(tread / uint(rusage_stop.Inblock-rusage_start.Inblock))
	}
	return
}
