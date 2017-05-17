package unpacker

import (
	"errors"
)

func (u *Unpacker) removeDevice(deviceId string) error {
	u.rwMutex.Lock()
	defer u.rwMutex.Unlock()
	defer u.updateUsageTimeWithLock()
	if device, ok := u.pState.Devices[deviceId]; !ok {
		return errors.New("unknown device ID: " + deviceId)
	} else {
		if device.StreamName != "" {
			return errors.New(
				"device ID: " + deviceId + " used by: " + device.StreamName)
		}
		delete(u.pState.Devices, deviceId)
		return u.writeStateWithLock()
	}
}
