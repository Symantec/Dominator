package unpacker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mbr"
)

var sysfsDirectory = "/sys/block"

func (u *Unpacker) addDevice(deviceId string) error {
	u.updateUsageTime()
	scannedDevices, err := scanDevices()
	if err != nil {
		return err
	}
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	defer u.updateUsageTimeWithLock()
	for device := range u.scannedDevices {
		delete(scannedDevices, device)
	}
	if len(scannedDevices) < 1 {
		return errors.New("no new devices found")
	}
	if len(scannedDevices) > 1 {
		return errors.New("too many new devices found")
	}
	var deviceName string
	for d := range scannedDevices {
		deviceName = d
	}
	device := deviceInfo{DeviceName: deviceName}
	if err := updateDeviceSize(&device); err != nil {
		return err
	}
	if err := partitionAndMkFs(deviceName, u.logger); err != nil {
		return err
	}
	u.pState.Devices[deviceId] = device
	return u.writeStateWithLock()
}

func checkIfBlockDevice(path string) error {
	if fi, err := os.Lstat(path); err != nil {
		return err
	} else if fi.Mode()&os.ModeType != os.ModeDevice {
		return fmt.Errorf("%s is not a device, mode: %s", path, fi.Mode())
	}
	return nil
}

func getPartition(devicePath string) (string, error) {
	partitionPaths := []string{devicePath + "1", devicePath + "p1"}
	for _, partitionPath := range partitionPaths {
		if err := checkIfBlockDevice(partitionPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if file, err := os.Open(partitionPath); err == nil {
			file.Close()
			return partitionPath, nil
		}
	}
	return "", fmt.Errorf("no partitions found for: %s", devicePath)
}

func partitionAndMkFs(deviceName string, logger log.DebugLogger) error {
	devicePath := "/dev/" + deviceName
	// Create a single partition. This is needed so that GRUB has a place to
	// live (between the MBR and first/only partition).
	logger.Printf("partitioning: %s\n", devicePath)
	if err := mbr.WriteDefault(devicePath, mbr.TABLE_TYPE_MSDOS); err != nil {
		return err
	}
	// udev has a bug where the partition device node is created and sometimes
	// is removed and then created again. Based on experiments the device node
	// is gone for ~15 milliseconds. Wait long enough to hopefully never
	// encounter this race again.
	logger.Println("sleeping 1s to work around udev race")
	time.Sleep(time.Second)
	partitionPath, err := getPartition(devicePath)
	if err != nil {
		return err
	}
	err = util.MakeExt4fs(partitionPath,
		fmt.Sprintf("rootfs@%x", time.Now().Unix()),
		[]string{"64bit", "metadata_csum"}, // TODO(rgooch): make this generic.
		8192, logger)
	if err != nil {
		return err
	}
	// Make sure it's still a block device. If not it means udev still had not
	// settled down after waiting, so remove the inode and return an error.
	if err := checkIfBlockDevice(partitionPath); err != nil {
		os.Remove(partitionPath)
		return err
	}
	return nil
}

func (u *Unpacker) prepareForAddDevice() error {
	scannedDevices, err := scanDevices()
	if err != nil {
		return err
	}
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	u.scannedDevices = scannedDevices
	return nil
}

func scanDevices() (map[string]struct{}, error) {
	file, err := os.Open(sysfsDirectory)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	names, err := file.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	scannedDevices := make(map[string]struct{})
	for _, name := range names {
		scannedDevices[name] = struct{}{}
	}
	return scannedDevices, nil
}

func updateDeviceSize(device *deviceInfo) error {
	deviceBlocks, err := readSysfsUint64(
		filepath.Join(sysfsDirectory, device.DeviceName, "size"))
	if err != nil {
		return err
	}
	device.size = deviceBlocks * 512
	return nil
}

func readSysfsUint64(filename string) (uint64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	var value uint64
	nScanned, err := fmt.Fscanf(file, "%d", &value)
	if err != nil {
		return 0, err
	}
	if nScanned < 1 {
		return 0, errors.New(fmt.Sprintf("only read %d values from: %s",
			nScanned, filename))
	}
	return value, nil
}
