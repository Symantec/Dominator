package fsbench

import (
	"github.com/Symantec/Dominator/lib/wsyscall"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"
)

const (
	BUFLEN      = 1024 * 1024
	MAX_TO_READ = 1024 * 1024 * 128
)

func openDirect(name string, flag int, perm os.FileMode) (file *os.File,
	err error) {
	return os.OpenFile(name, flag|syscall.O_DIRECT, perm)
}

func GetDevnumForFile(name string) (devnum uint64, err error) {
	var stat wsyscall.Stat_t
	if err = wsyscall.Stat(name, &stat); err != nil {
		return 0, err
	}
	return stat.Dev, nil
}

func getDevnumForDevice(name string) (devnum uint64, err error) {
	var stat wsyscall.Stat_t
	if err = wsyscall.Stat(name, &stat); err != nil {
		return 0, err
	}
	return stat.Rdev, nil
}

func getDevnodeForFile(name string) (devnode string, err error) {
	var devnum uint64
	devnum, err = GetDevnumForFile(name)
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

// Compute the maximum read speed of a block device, given a file within a
// file-system mounted on the block device.
// If I/O accounting is enabled, blocksPerSecond will be non-zero.
func GetReadSpeed(name string) (bytesPerSecond, blocksPerSecond uint64,
	err error) {
	bytesPerSecond = 0
	blocksPerSecond = 0
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
	defer file.Close()
	var tread uint = 0
	buffer := make([]byte, BUFLEN)
	var rusage_start, rusage_stop syscall.Rusage
	if err = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage_start); err != nil {
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
	elapsed := time.Since(time_start)
	bytesPerSecond = uint64(float64(tread) / elapsed.Seconds())
	if err = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage_stop); err != nil {
		return
	}
	if rusage_stop.Inblock > rusage_start.Inblock {
		blocksPerSecond = uint64(float64(rusage_stop.Inblock-
			rusage_start.Inblock) / elapsed.Seconds())
	}
	return
}
