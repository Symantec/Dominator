package json

import (
	"io"
)

func WriteWithIndent(w io.Writer, indent string, value interface{}) error {
	return writeWithIndent(w, indent, value)
}
