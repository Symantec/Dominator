package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/Symantec/Dominator/lib/concurrent"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectcache"
)

func newObjectServer(baseDir string, logger log.Logger) (
	*ObjectServer, error) {
	fi, err := os.Stat(baseDir)
	if err != nil {
		return nil, errors.New(
			fmt.Sprintf("Cannot stat: %s: %s\n", baseDir, err))
	}
	if !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory\n", baseDir))
	}
	objSrv := &ObjectServer{
		baseDir:               baseDir,
		logger:                logger,
		sizesMap:              make(map[hash.Hash]uint64),
		lastGarbageCollection: time.Now(),
		lastMutationTime:      time.Now(),
	}
	state := concurrent.NewState(0)
	startTime := time.Now()
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	if err = scanDirectory(objSrv, baseDir, "", state); err != nil {
		return nil, err
	}
	if err := state.Reap(); err != nil {
		return nil, err
	}
	plural := ""
	if len(objSrv.sizesMap) != 1 {
		plural = "s"
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	userTime := time.Duration(rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	logger.Printf("Scanned %d object%s in %s (%s user CPUtime)\n",
		len(objSrv.sizesMap), plural, time.Since(startTime), userTime)
	return objSrv, nil
}

func scanDirectory(objSrv *ObjectServer, baseDir string, subpath string,
	state *concurrent.State) error {
	myPathName := path.Join(baseDir, subpath)
	file, err := os.Open(myPathName)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if len(name) > 0 && name[0] == '.' {
			continue // Skip hidden paths.
		}
		fullPathName := path.Join(myPathName, name)
		fi, err := os.Lstat(fullPathName)
		if err != nil {
			continue
		}
		filename := path.Join(subpath, name)
		if fi.IsDir() {
			if state == nil {
				if err := scanDirectory(objSrv, baseDir, filename,
					nil); err != nil {
					return err
				}
			} else {
				// GoRun() cannot be used recursively, so limit concurrency to
				// the top level. It's also more efficient this way.
				if err := state.GoRun(func() error {
					return scanDirectory(objSrv, baseDir, filename, nil)
				}); err != nil {
					return err
				}
			}
		} else {
			if fi.Size() < 1 {
				return errors.New(
					fmt.Sprintf("zero-length file: %s", fullPathName))
			}
			hash, err := objectcache.FilenameToHash(filename)
			if err != nil {
				return err
			}
			objSrv.rwLock.Lock()
			objSrv.sizesMap[hash] = uint64(fi.Size())
			objSrv.rwLock.Unlock()
		}
	}
	return nil
}
