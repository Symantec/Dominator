package unpacker

import (
	"bufio"
	"encoding/json"
	"github.com/Symantec/Dominator/lib/srpc"
	"log"
	"os"
	"path"
	"syscall"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
)

func load(baseDir string, imageServerAddress string, logger *log.Logger) (
	*Unpacker, error) {
	if err := os.MkdirAll(path.Join(baseDir, "mnt"), dirPerms); err != nil {
		return nil, err
	}
	u := &Unpacker{
		baseDir:             baseDir,
		imageServerResource: srpc.NewClientResource("tcp", imageServerAddress),
		logger:              logger,
	}
	file, err := os.Open(path.Join(baseDir, stateFile))
	if err != nil {
		if os.IsNotExist(err) {
			u.pState.Devices = make(map[string]deviceInfo)
			u.pState.ImageStreams = make(map[string]*imageStreamInfo)
			return u, nil
		}
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(bufio.NewReader(file))
	if err := decoder.Decode(&u.pState); err != nil {
		return nil, err
	}
	// Fill in sizes.
	for deviceName, device := range u.pState.Devices {
		if err := updateDeviceSize(&device); err != nil {
			return nil, err
		}
		u.pState.Devices[deviceName] = device
	}
	// Set up streams.
	for streamName := range u.pState.ImageStreams {
		if _, err := u.setupStream(streamName); err != nil {
			return nil, err
		}
	}
	return u, nil
}
