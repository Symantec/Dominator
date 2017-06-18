package fsutil

import (
	"os"
)

func createRenamingWriter(filename string, perm os.FileMode) (
	*RenamingWriter, error) {
	writer := &RenamingWriter{filename: filename}
	tmpFilename := filename + "~"
	var err error
	writer.File, err = os.OpenFile(tmpFilename, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *RenamingWriter) close() error {
	tmpFilename := w.filename + "~"
	defer os.Remove(tmpFilename)
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
