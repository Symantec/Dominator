package json

import (
	"encoding/json"
	"io"
)

func writeWithIndent(w io.Writer, indent string, value interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", indent)
	return encoder.Encode(value)
}
