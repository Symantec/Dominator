package fsutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func copyToFile(destFilename string, perm os.FileMode, reader io.Reader,
	length uint64) error {
	tmpFilename := destFilename + "~"
	destFile, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFilename)
	defer destFile.Close()
	writer := bufio.NewWriter(destFile)
	defer writer.Flush()
	var nCopied int64
	if nCopied, err = io.Copy(writer, reader); err != nil {
		return fmt.Errorf("error copying: %s", err)
	}
	if nCopied != int64(length) {
		return fmt.Errorf("expected length: %d, got: %d for: %s\n",
			length, nCopied, tmpFilename)
	}
	return os.Rename(tmpFilename, destFilename)
}
