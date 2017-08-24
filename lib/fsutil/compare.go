package fsutil

import (
	"bytes"
	"io"
	"os"
)

func compareFiles(leftFilename, rightFilename string) (bool, error) {
	leftFile, err := os.Open(leftFilename)
	if err != nil {
		return false, err
	}
	defer leftFile.Close()
	leftFI, err := leftFile.Stat()
	if err != nil {
		return false, err
	}
	rightFile, err := os.Open(rightFilename)
	if err != nil {
		return false, err
	}
	defer rightFile.Close()
	rightFI, err := rightFile.Stat()
	if err != nil {
		return false, err
	}
	if leftFI.Size() != rightFI.Size() {
		return false, nil
	}
	leftBuffer := make([]byte, 65536)
	rightBuffer := make([]byte, 65536)
	for {
		nLeft, err := io.ReadFull(leftFile, leftBuffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return false, err
		}
		nRight, err := io.ReadFull(rightFile, rightBuffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return false, err
		}
		if nLeft != nRight {
			return false, nil
		}
		if nLeft == 0 {
			break
		}
		if bytes.Compare(leftBuffer[:nLeft], rightBuffer[:nRight]) != 0 {
			return false, nil
		}
	}
	return true, nil
}
