package fsutil

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

func CopyToFile(destFilename string, reader io.Reader, length int64) error {
	tmpFilename := destFilename + "~"
	destFile, err := os.Create(tmpFilename)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFilename)
	defer destFile.Close()
	writer := bufio.NewWriter(destFile)
	defer writer.Flush()
	var nCopied int64
	if nCopied, err = io.Copy(writer, reader); err != nil {
		return errors.New(fmt.Sprintf("error copying: %s", err.Error()))
	}
	if nCopied != length {
		return errors.New(fmt.Sprintf("expected length: %d, got: %d for: %s\n",
			length, nCopied, tmpFilename))
	}
	return os.Rename(tmpFilename, destFilename)
}
