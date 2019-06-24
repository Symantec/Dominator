package fsutil

import (
	"os"
	"time"
)

var fsyncSemaphore = make(chan struct{}, 1)

func createRenamingWriter(filename string, perm os.FileMode) (
	*RenamingWriter, error) {
	writer := &RenamingWriter{filename: filename}
	tmpFilename := filename + "~"
	var err error
	writer.File, err = os.OpenFile(tmpFilename,
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return nil, err
	}
	return writer, nil
}

// fsyncFile will call file.Sync if it has not been called recently. This
// attempts to reduce the performance problems of fsync(2) by potentially
// sacrificing some file-system consistency.
func fsyncFile(file *os.File) error {
	select {
	case fsyncSemaphore <- struct{}{}:
	default:
		return nil
	}
	startTime := time.Now()
	err := file.Sync()
	duration := time.Since(startTime)
	go func() {
		time.Sleep(duration)
		<-fsyncSemaphore
	}()
	return err
}

func (w *RenamingWriter) close() error {
	tmpFilename := w.filename + "~"
	defer os.Remove(tmpFilename)
	if !w.abort {
		if err := fsyncFile(w.File); err != nil {
			return err
		}
	}
	if err := w.File.Close(); err != nil {
		return err
	}
	if w.abort {
		return nil
	}
	return os.Rename(tmpFilename, w.filename)
}

func (w *RenamingWriter) write(p []byte) (int, error) {
	if nWritten, err := w.File.Write(p); err != nil {
		w.abort = true
		return nWritten, err
	} else {
		return nWritten, nil
	}
}
