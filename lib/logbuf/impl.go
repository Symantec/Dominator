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
	"strings"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/bufwriter"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
	"github.com/Cloud-Foundations/Dominator/lib/format"
)

const (
	dirPerms = syscall.S_IRWXU | syscall.S_IRGRP | syscall.S_IXGRP |
		syscall.S_IROTH | syscall.S_IXOTH
	filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
		syscall.S_IROTH

	reopenMessage = "Closing and will open new logfile"
	timeLayout    = "2006-01-02:15:04:05.999"
)

func newLogBuffer(options Options) *LogBuffer {
	if options.MaxBufferLines < 100 {
		options.MaxBufferLines = 100
	}
	if options.MaxFileSize < 16384 {
		options.MaxFileSize = 16384
	}
	if options.Quota < 65536 {
		options.Quota = 65536
	}
	if options.AlsoLogToStderr {
		options.RedirectStderr = false // Prevent duplicates sent to stderr.
	}
	logBuffer := &LogBuffer{
		options: options,
		buffer:  ring.New(int(options.MaxBufferLines)),
	}
	if err := logBuffer.setupFileLogging(); err != nil {
		fmt.Fprintln(logBuffer, err)
	}
	logBuffer.addHttpHandlers()
	return logBuffer
}

func (lb *LogBuffer) setupFileLogging() error {
	if lb.options.Directory == "" {
		return nil
	}
	if err := lb.createLogDirectory(); err != nil {
		return err
	}
	writeNotifier := make(chan struct{}, 1)
	lb.writeNotifier = writeNotifier
	go lb.idleMarker(writeNotifier)
	return nil
}

func (lb *LogBuffer) createLogDirectory() error {
	if fi, err := os.Stat(lb.options.Directory); err != nil {
		if err := os.Mkdir(lb.options.Directory, dirPerms); err != nil {
			return fmt.Errorf("error creating: %s: %s",
				lb.options.Directory, err)
		}
	} else if !fi.IsDir() {
		return errors.New(lb.options.Directory + ": is not a directory")
	}
	lb.scanPreviousForPanic()
	return lb.enforceQuota()
}

func (lb *LogBuffer) scanPreviousForPanic() {
	target, err := os.Readlink(path.Join(lb.options.Directory, "latest"))
	if err != nil {
		return
	}
	targetPath := path.Join(lb.options.Directory, target)
	file, err := os.Open(targetPath)
	if err != nil {
		return
	}
	go func() {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "panic: ") {
				lb.rwMutex.Lock()
				lb.panicLogfile = &target
				lb.rwMutex.Unlock()
				if fi, err := os.Stat(targetPath); err != nil {
					return
				} else {
					os.Chmod(targetPath, fi.Mode()|os.ModeSticky)
				}
				return
			}
		}
	}()
}

func (lb *LogBuffer) dump(writer io.Writer, prefix, postfix string,
	recentFirst bool) error {
	entries := lb.getEntries()
	if recentFirst {
		reverseEntries(entries)
	}
	for _, entry := range entries {
		writer.Write([]byte(prefix))
		writer.Write(entry)
		writer.Write([]byte(postfix))
	}
	return nil
}

func (lb *LogBuffer) flush() error {
	lb.rwMutex.Lock()
	defer lb.rwMutex.Unlock()
	if lb.writer != nil {
		return lb.writer.Flush()
	}
	return nil
}

func (lb *LogBuffer) write(p []byte) (n int, err error) {
	if lb.options.AlsoLogToStderr {
		os.Stderr.Write(p)
	}
	val := make([]byte, len(p))
	copy(val, p)
	lb.rwMutex.Lock()
	sendNotify := lb.writeToLogFile(p)
	lb.buffer.Value = val
	lb.buffer = lb.buffer.Next()
	lb.rwMutex.Unlock()
	if sendNotify {
		lb.writeNotifier <- struct{}{}
	}
	return len(p), nil
}

// This should be called with the lock held.
func (lb *LogBuffer) writeToLogFile(p []byte) bool {
	if lb.writer == nil {
		return false
	}
	lb.writer.Write(p)
	lb.fileSize += flagutil.Size(len(p))
	if lb.fileSize > lb.options.MaxFileSize {
		lb.closeAndOpenNewFile()
	}
	lb.usage += flagutil.Size(len(p))
	if lb.usage > lb.options.Quota {
		lb.enforceQuota()
	}
	return true
}

// This should be called with the lock held.
func (lb *LogBuffer) closeAndOpenNewFile() error {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	nWritten, _ := fmt.Fprintf(lb.writer, "%d/%02d/%02d %02d:%02d:%02d %s\n",
		year, month, day, hour, minute, second, reopenMessage)
	lb.usage += flagutil.Size(nWritten)
	lb.writer.Flush()
	lb.writer = nil
	lb.file.Close()
	if err := lb.openNewFile(); err != nil {
		return err
	}
	return nil
}

// This should be called with the lock held.
func (lb *LogBuffer) openNewFile() error {
	lb.fileSize = 0
	filename := time.Now().Format(timeLayout)
	file, err := os.OpenFile(path.Join(lb.options.Directory, filename),
		os.O_CREATE|os.O_WRONLY, filePerms)
	if err != nil {
		return err
	}
	if lb.options.RedirectStderr {
		syscall.Dup2(int(file.Fd()), int(os.Stdout.Fd()))
		syscall.Dup2(int(file.Fd()), int(os.Stderr.Fd()))
	}
	lb.file = file
	lb.writer = bufwriter.NewWriter(file, time.Second)
	symlink := path.Join(lb.options.Directory, "latest")
	tmpSymlink := symlink + "~"
	os.Remove(tmpSymlink)
	os.Symlink(filename, tmpSymlink)
	return os.Rename(tmpSymlink, symlink)
}

// This should be called with the lock held.
func (lb *LogBuffer) enforceQuota() error {
	file, err := os.Open(lb.options.Directory)
	if err != nil {
		return err
	}
	names, err := file.Readdirnames(-1)
	file.Close()
	if err != nil {
		return err
	}
	sort.Strings(names)
	var numBytesDeleted, numFilesDeleted uint64
	var usage flagutil.Size
	deletedLatestFile := false
	deleteRemainingFiles := false
	latestFile := true
	for index := len(names) - 1; index >= 0; index-- {
		filename := path.Join(lb.options.Directory, names[index])
		fi, err := os.Lstat(filename)
		if err == os.ErrNotExist {
			continue
		}
		if err != nil {
			return err
		}
		if fi.Mode().IsRegular() {
			size := flagutil.Size(fi.Size())
			if size < lb.options.Quota>>10 {
				size = lb.options.Quota >> 10 // Limit number of files to 1024.
			}
			if size+usage > lb.options.Quota || deleteRemainingFiles {
				os.Remove(filename)
				numBytesDeleted += uint64(fi.Size())
				numFilesDeleted++
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
		if err := lb.openNewFile(); err != nil {
			return err
		}
	}
	if numBytesDeleted > 0 {
		now := time.Now()
		year, month, day := now.Date()
		hour, minute, second := now.Clock()
		nWritten, _ := fmt.Fprintf(lb.writer,
			"%d/%02d/%02d %02d:%02d:%02d Deleted %s in %d files\n",
			year, month, day, hour, minute, second,
			format.FormatBytes(numBytesDeleted), numFilesDeleted)
		lb.fileSize += flagutil.Size(nWritten)
		lb.usage += flagutil.Size(nWritten)
	}
	return nil
}

func (lb *LogBuffer) idleMarker(writeNotifier <-chan struct{}) {
	idleMarkDuration := lb.options.IdleMarkTimeout
	if idleMarkDuration < 1 {
		for {
			<-writeNotifier
		}
	}
	idleMarkTimer := time.NewTimer(idleMarkDuration)
	for {
		select {
		case <-writeNotifier:
			idleMarkTimer.Reset(idleMarkDuration)
		case <-idleMarkTimer.C:
			lb.writeMark()
			idleMarkTimer.Reset(idleMarkDuration)
		}
	}
}

func (lb *LogBuffer) getEntries() [][]byte {
	lb.rwMutex.RLock()
	defer lb.rwMutex.RUnlock()
	entries := make([][]byte, 0, lb.buffer.Len())
	lb.buffer.Do(func(p interface{}) {
		if p != nil {
			entries = append(entries, p.([]byte))
		}
	})
	return entries
}

func (lb *LogBuffer) dumpSince(writer io.Writer, name string,
	earliestTime time.Time, prefix, postfix string, recentFirst bool) (
	bool, error) {
	file, err := os.Open(path.Join(lb.options.Directory,
		path.Base(path.Clean(name))))
	if err != nil {
		return false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	timeFormat := "2006/01/02 15:04:05"
	minLength := len(timeFormat) + 2
	foundReopenMessage := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, reopenMessage) {
			foundReopenMessage = true
			continue
		}
		if len(line) >= minLength {
			timeString := line[:minLength-2]
			timeStamp, err := time.ParseInLocation(timeFormat, timeString,
				time.Local)
			if err == nil && timeStamp.Before(earliestTime) {
				continue
			}
		}
		if recentFirst {
			lines = append(lines, line)
		} else {
			writer.Write([]byte(prefix))
			writer.Write([]byte(line))
			writer.Write([]byte(postfix))
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if recentFirst {
		reverseStrings(lines)
		for _, line := range lines {
			writer.Write([]byte(prefix))
			writer.Write([]byte(line))
			writer.Write([]byte(postfix))
		}
	}
	return foundReopenMessage, nil
}

func (lb *LogBuffer) writeMark() {
	now := time.Now()
	year, month, day := now.Date()
	hour, minute, second := now.Clock()
	str := fmt.Sprintf("%d/%02d/%02d %02d:%02d:%02d MARK\n",
		year, month, day, hour, minute, second)
	lb.rwMutex.Lock()
	defer lb.rwMutex.Unlock()
	lb.writeToLogFile([]byte(str))
}

func reverseEntries(entries [][]byte) {
	length := len(entries)
	for index := 0; index < length/2; index++ {
		entries[index], entries[length-1-index] =
			entries[length-1-index], entries[index]
	}
}

func reverseStrings(strings []string) {
	length := len(strings)
	for index := 0; index < length/2; index++ {
		strings[index], strings[length-1-index] =
			strings[length-1-index], strings[index]
	}
}

func init() {
	UseFlagSet(flag.CommandLine)
}
