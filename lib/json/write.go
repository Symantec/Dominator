package json

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

func writeToFile(filename string, perm os.FileMode, indent string,
	value interface{}) error {
	tmpFilename := filename + "~"
	file, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		perm)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFilename)
	writer := bufio.NewWriter(file)
	if err := writeWithIndent(writer, indent, value); err != nil {
		return err
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmpFilename, filename)
}

func writeWithIndent(w io.Writer, indent string, value interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", indent)
	return encoder.Encode(value)
}
