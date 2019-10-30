package fsbench

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/wsyscall"
)

const (
	BUFLEN      = 1024 * 1024
	MAX_TO_READ = 1024 * 1024 * 128
)

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

func getDevnodeForFile(name string) (string, error) {
	devnum, err := GetDevnumForFile(name)
	if err != nil {
		return "", err
	}
	fi_list, err := ioutil.ReadDir("/dev")
	if err != nil {
		return "", fmt.Errorf("error reading directory: /dev: %s", err)
	}
	for _, fi := range fi_list {
		if (fi.Mode()&os.ModeDevice != 0) &&
			(fi.Mode()&os.ModeCharDevice == 0) {
			devpath := path.Join("/dev", fi.Name())
			dnum, err := getDevnumForDevice(devpath)
			if err != nil {
				return "", err
			}
			if dnum == devnum {
				return devpath, nil
			}
		}
	}
	// Can't find it the traditional way. btrfs is a known culprit (since it
	// can be on multiple devices, it claims a virtual device). Read
	// /proc/mounts instead. First walk up the tree to find the mount point.
	mountPoint := name
	for {
		parent := path.Dir(mountPoint)
		parentDevnum, err := GetDevnumForFile(parent)
		if err != nil {
			return "", err
		}
		if parentDevnum != devnum {
			break
		}
		mountPoint = parent
		if parent == "/" {
			break
		}
	}
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[1] == mountPoint {
			if strings.HasPrefix(fields[0], "/dev/") {
				return fields[0], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("unable to find device path for: %s", name)
}

// Compute the maximum read speed of a block device, given a file within a
// file-system mounted on the block device.
// Returns: bytesPerSecond, blocksPerSecond, error
// If I/O accounting is enabled, blocksPerSecond will be non-zero.
func GetReadSpeed(name string) (uint64, uint64, error) {
	devpath, err := getDevnodeForFile(name)
	if err != nil {
		return 0, 0, err
	}
	file, err := openDirect(devpath, os.O_RDONLY, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("error opening: %s: %s", devpath, err)
	}
	defer file.Close()
	var tread uint = 0
	buffer := make([]byte, BUFLEN)
	var rusage_start, rusage_stop syscall.Rusage
	if err = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage_start); err != nil {
		return 0, 0, fmt.Errorf("error getting resource usage: %s", err)
	}
	time_start := time.Now()
	for tread < MAX_TO_READ {
		var nread int
		nread, err = file.Read(buffer)
		tread += uint(nread)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, 0, fmt.Errorf("error reading: %s: %s", devpath, err)
		}
	}
	elapsed := time.Since(time_start)
	bytesPerSecond := uint64(float64(tread) / elapsed.Seconds())
	if err = syscall.Getrusage(syscall.RUSAGE_SELF, &rusage_stop); err != nil {
		return 0, 0, fmt.Errorf("error getting resource usage: %s", err)
	}
	var blocksPerSecond uint64
	if rusage_stop.Inblock > rusage_start.Inblock {
		blocksPerSecond = uint64(float64(rusage_stop.Inblock-
			rusage_start.Inblock) / elapsed.Seconds())
	}
	return bytesPerSecond, blocksPerSecond, nil
}
