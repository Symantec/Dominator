package mdb

import (
	"bytes"
	"encoding/json"
	"io"
)

func (mdb *Mdb) debugWrite(w io.Writer) error {
	b, _ := json.Marshal(mdb)
	var out bytes.Buffer
	json.Indent(&out, b, "", "    ")
	_, err := out.WriteTo(w)
	return err
}
