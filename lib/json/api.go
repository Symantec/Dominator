package json

import (
	"io"
)

func ReadFromFile(filename string, value interface{}) error {
	return readFromFile(filename, value)
}

func WriteWithIndent(w io.Writer, indent string, value interface{}) error {
	return writeWithIndent(w, indent, value)
}
