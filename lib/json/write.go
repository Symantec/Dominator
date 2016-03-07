package json

import (
	"bytes"
	"encoding/json"
	"io"
)

func writeWithIndent(w io.Writer, indent string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", indent); err != nil {
		return err
	}
	_, err = out.WriteTo(w)
	return err
}
