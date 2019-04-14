package fsutil

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"
)

func TestWaitFile(t *testing.T) {
	dirname, err := ioutil.TempDir("", "WaitFileTests")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dirname)
	pathNotExist := path.Join(dirname, "never-exists")
	rc, err := WaitFile(pathNotExist, time.Microsecond)
	if err == nil {
		t.Errorf("Expected timeout error for non-existent file")
		rc.Close()
	}
	pathExists := path.Join(dirname, "exists")
	file, err := os.Create(pathExists)
	if err != nil {
		t.Error(err)
	}
	file.Close()
	rc, err = WaitFile(pathExists, time.Microsecond)
	if err != nil {
		t.Error(err)
	} else {
		rc.Close()
	}
	pathExistsLater := path.Join(dirname, "exists-later")
	go func() {
		time.Sleep(time.Millisecond * 50)
		file, err := os.Create(pathExistsLater)
		if err != nil {
			t.Error(err)
			return
		}
		file.Close()
	}()
	rc, err = WaitFile(pathExistsLater, time.Millisecond*10)
	if err == nil {
		rc.Close()
		t.Errorf("Expected timeout error for non-existent file")
	}
	rc, err = WaitFile(pathExistsLater, time.Millisecond*90)
	if err != nil {
		t.Error(err)
	} else {
		rc.Close()
	}
}
