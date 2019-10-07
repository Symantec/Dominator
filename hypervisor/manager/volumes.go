package manager

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/Symantec/Dominator/lib/format"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const (
	procMounts    = "/proc/mounts"
	sysClassBlock = "/sys/class/block"
)

type mountInfo struct {
	mountPoint string
	size       uint64
}

func demapDevice(device string) (string, error) {
	sysDir := filepath.Join(sysClassBlock, filepath.Base(device), "slaves")
	if file, err := os.Open(sysDir); err != nil {
		return device, nil
	} else {
		defer file.Close()
		names, err := file.Readdirnames(-1)
		if err != nil {
			return "", err
		}
		if len(names) != 1 {
			return "", fmt.Errorf("%s has %d entries", device, len(names))
		}
		return filepath.Join("/dev", names[0]), nil
	}
}

func getFreeSpace(dirname string, freeSpaceTable map[string]uint64) (
	uint64, error) {
	if freeSpace, ok := freeSpaceTable[dirname]; ok {
		return freeSpace, nil
	}
	var statbuf syscall.Statfs_t
	if err := syscall.Statfs(dirname, &statbuf); err != nil {
		return 0, fmt.Errorf("error statfsing: %s: %s", dirname, err)
	}
	freeSpace := uint64(statbuf.Bfree * uint64(statbuf.Bsize))
	freeSpaceTable[dirname] = freeSpace
	return freeSpace, nil
}

func getMounts() (map[string]string, error) {
	file, err := os.Open(procMounts)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	mounts := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		var device, mountPoint, fsType, fsOptions, junk string
		nScanned, err := fmt.Sscanf(line, "%s %s %s %s %s",
			&device, &mountPoint, &fsType, &fsOptions, &junk)
		if err != nil {
			return nil, err
		}
		if nScanned < 4 {
			return nil, errors.New(fmt.Sprintf("only read %d values from %s",
				nScanned, line))
		}
		if mountPoint == "/boot" {
			continue
		}
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}
		if device == "/dev/root" { // Ignore this dumb shit.
			continue
		}
		if target, err := filepath.EvalSymlinks(device); err != nil {
			return nil, err
		} else {
			device = target
		}
		device, err = demapDevice(device)
		if err != nil {
			return nil, err
		}
		device = device[5:]
		if _, ok := mounts[device]; !ok { // Pick the first mount point.
			mounts[device] = mountPoint
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return mounts, nil
}

func getVolumeDirectories() ([]string, error) {
	mounts, err := getMounts()
	if err != nil {
		return nil, err
	}
	var mountPointsToUse []string
	biggestMounts := make(map[string]mountInfo)
	for device, mountPoint := range mounts {
		sysDir := filepath.Join(sysClassBlock, device)
		linkTarget, err := os.Readlink(sysDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		_, err = os.Stat(filepath.Join(sysDir, "partition"))
		if err != nil {
			if os.IsNotExist(err) { // Not a partition: easy!
				mountPointsToUse = append(mountPointsToUse, mountPoint)
				continue
			}
			return nil, err
		}
		var statbuf syscall.Statfs_t
		if err := syscall.Statfs(mountPoint, &statbuf); err != nil {
			return nil, fmt.Errorf("error statfsing: %s: %s", mountPoint, err)
		}
		size := uint64(statbuf.Blocks * uint64(statbuf.Bsize))
		parentDevice := filepath.Base(filepath.Dir(linkTarget))
		if biggestMount, ok := biggestMounts[parentDevice]; !ok {
			biggestMounts[parentDevice] = mountInfo{mountPoint, size}
		} else if size > biggestMount.size {
			biggestMounts[parentDevice] = mountInfo{mountPoint, size}
		}
	}
	for _, biggestMount := range biggestMounts {
		mountPointsToUse = append(mountPointsToUse, biggestMount.mountPoint)
	}
	var volumeDirectories []string
	for _, mountPoint := range mountPointsToUse {
		volumeDirectories = append(volumeDirectories,
			filepath.Join(mountPoint, "hyper-volumes"))
	}
	sort.Strings(volumeDirectories)
	return volumeDirectories, nil
}

func (m *Manager) findFreeSpace(size uint64, freeSpaceTable map[string]uint64,
	position *int) (string, error) {
	if *position >= len(m.volumeDirectories) {
		*position = 0
	}
	startingPosition := *position
	for {
		freeSpace, err := getFreeSpace(m.volumeDirectories[*position],
			freeSpaceTable)
		if err != nil {
			return "", err
		}
		if size < freeSpace {
			dirname := m.volumeDirectories[*position]
			freeSpaceTable[dirname] -= size
			return dirname, nil
		}
		*position++
		if *position >= len(m.volumeDirectories) {
			*position = 0
		}
		if *position == startingPosition {
			return "", fmt.Errorf("not enough free space for %s volume",
				format.FormatBytes(size))
		}
	}
}

func (m *Manager) getVolumeDirectories(rootSize uint64,
	volumes []proto.Volume, spreadVolumes bool) ([]string, error) {
	sizes := make([]uint64, 1, len(volumes)+1)
	sizes[0] = rootSize
	for _, volume := range volumes {
		if volume.Size > 0 {
			sizes = append(sizes, volume.Size)
		}
	}
	freeSpaceTable := make(map[string]uint64, len(m.volumeDirectories))
	directoriesToUse := make([]string, 0, len(sizes))
	position := 0
	for len(sizes) > 0 {
		dirname, err := m.findFreeSpace(sizes[0], freeSpaceTable, &position)
		if err != nil {
			return nil, err
		}
		directoriesToUse = append(directoriesToUse, dirname)
		sizes = sizes[1:]
		if spreadVolumes {
			position++
		}
	}
	return directoriesToUse, nil
}
