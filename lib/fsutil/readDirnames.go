package fsutil

import "os"

func readDirnames(dirname string, ignoreMissing bool) ([]string, error) {
	if file, err := os.Open(dirname); err != nil {
		if ignoreMissing && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	} else {
		defer file.Close()
		dirnames, err := file.Readdirnames(-1)
		return dirnames, err
	}
}
