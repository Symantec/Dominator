package unpacker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	// Create a single partition. This is needed so that GRUB has a place to
	// live (between the MBR and first/only partition).
	devicePath := filepath.Join("/dev", deviceName)
	u.logger.Printf("partitioning: %s\n", devicePath)
	if err := mbr.WriteDefault(devicePath, mbr.TABLE_TYPE_MSDOS); err != nil {
		return err
	}
	device.partitionTimestamp = time.Now()
	u.pState.Devices[deviceId] = device
	return u.writeStateWithLock()
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
