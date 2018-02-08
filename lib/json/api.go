package json

import (
	"io"
	"os"
)

func ReadFromFile(filename string, value interface{}) error {
	return readFromFile(filename, value)
}

func WriteToFile(filename string, perm os.FileMode, indent string,
	value interface{}) error {
	return writeToFile(filename, perm, indent, value)
}

func WriteWithIndent(w io.Writer, indent string, value interface{}) error {
	return writeWithIndent(w, indent, value)
}
