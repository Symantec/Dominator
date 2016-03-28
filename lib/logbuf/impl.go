package logbuf

import (
	"bufio"
	"container/ring"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"syscall"
	"time"
)

const (
	dirPerms  = syscall.S_IRWXU
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP
)

var (
	alsoLogToStderr = flag.Bool("alsoLogToStderr", false,
		"If true, also write logs to stderr")
	logDir = flag.String("logDir", path.Join("/var/log", path.Base(os.Args[0])),
		"Directory to write log data to")
	logQuota = flag.Uint("logQuota", 10,
		"Log quota in MiB. If exceeded, old logs are deleted")
)

func newLogBuffer(length uint, dirname string, quota uint64) *LogBuffer {
	logBuffer := &LogBuffer{
		buffer: ring.New(int(length)),
		logDir: dirname,
		quota:  quota}
	if err := logBuffer.setupFileLogging(); err != nil {
		fmt.Fprintln(logBuffer, err)
	}
	return logBuffer
}

func (lb *LogBuffer) setupFileLogging() error {
	if lb.logDir == "" {
		return nil
	}
	if err := lb.createLogDirectory(); err != nil {
		return err
	}
	writeNotifier := make(chan struct{}, 1)
	lb.writeNotifier = writeNotifier
	go lb.flushWhenIdle(writeNotifier)
	return nil
}

func (lb *LogBuffer) createLogDirectory() error {
	if fi, err := os.Stat(lb.logDir); err != nil {
		if err := os.Mkdir(lb.logDir, dirPerms); err != nil {
			return fmt.Errorf("error creating: %s: %s", lb.logDir, err)
		}
		fi, err = os.Stat(lb.logDir)
	} else if !fi.IsDir() {
		return errors.New(lb.logDir + ": is not a directory")
	}
	return lb.enforceQuota()
}

func (lb *LogBuffer) write(p []byte) (n int, err error) {
	if *alsoLogToStderr {
		os.Stderr.Write(p)
	}
	lb.rwMutex.Lock()
	defer lb.rwMutex.Unlock()
	lb.writeToLogFile(p)
	val := make([]byte, len(p))
	copy(val, p)
	lb.buffer.Value = val
	lb.buffer = lb.buffer.Next()
	return len(p), nil
}

// This should be called with the lock held.
func (lb *LogBuffer) writeToLogFile(p []byte) {
	if lb.writer == nil {
		return
	}
	lb.writer.Write(p)
	lb.writeNotifier <- struct{}{}
	lb.usage += uint64(len(p))
	if lb.usage <= lb.quota {
		return
	}
	lb.enforceQuota()
}

// This should be called with the lock held.
func (lb *LogBuffer) enforceQuota() error {
	file, err := os.Open(lb.logDir)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	sort.Strings(names)
	var usage uint64
	deletedLatestFile := false
	deleteRemainingFiles := false
	latestFile := true
	for index := len(names) - 1; index >= 0; index-- {
		filename := path.Join(lb.logDir, names[index])
		fi, err := os.Lstat(filename)
		if err == os.ErrNotExist {
			continue
		}
		if err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			size := uint64(fi.Size())
			if size+usage > lb.quota || deleteRemainingFiles {
				os.Remove(filename)
				deleteRemainingFiles = true
				if latestFile {
					deletedLatestFile = true
				}
			} else {
				usage += size
			}
			latestFile = false
		}
	}
	lb.usage = usage
	if deletedLatestFile && lb.file != nil {
		lb.writer.Flush()
		lb.writer = nil
		lb.file.Close()
		lb.file = nil
	}
	if lb.file == nil {
		now := time.Now()
		filename := fmt.Sprintf("%d%02d%02d:%02d%02d%02d.%03d",
			now.Year(), now.Month(), now.Day(),
			now.Hour(), now.Minute(), now.Second(), now.Nanosecond()/1000000)
		file, err := os.OpenFile(path.Join(lb.logDir, filename),
			os.O_CREATE|os.O_WRONLY, filePerms)
		if err != nil {
			return err
		}
		lb.file = file
		lb.writer = bufio.NewWriter(file)
		symlink := path.Join(lb.logDir, "latest")
		os.Symlink(filename, symlink+"~")
		os.Rename(symlink+"~", symlink)
	}
	return nil
}

func (lb *LogBuffer) flushWhenIdle(writeNotifier <-chan struct{}) {
	timer := time.NewTimer(time.Second)
	for {
		select {
		case <-writeNotifier:
			timer.Reset(time.Second)
		case <-timer.C:
			lb.writer.Flush()
		}
	}
}

func (lb *LogBuffer) dump(writer io.Writer, prefix, postfix string) error {
	lb.rwMutex.RLock()
	defer lb.rwMutex.RUnlock()
	lb.buffer.Do(func(p interface{}) {
		if p != nil {
			writer.Write([]byte(prefix))
			writer.Write(p.([]byte))
			writer.Write([]byte(postfix))
		}
	})
	return nil
}

func (lb *LogBuffer) writeHtml(writer io.Writer) {
	fmt.Fprintln(writer, "Logs:<br>")
	fmt.Fprintln(writer, "<pre>")
	lb.Dump(writer, "", "")
	fmt.Fprintln(writer, "</pre>")
}
