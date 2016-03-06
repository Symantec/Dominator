package mdb

import (
	"github.com/Symantec/Dominator/lib/json"
	"io"
)

func (mdb *Mdb) debugWrite(w io.Writer) error {
	return json.WriteWithIndent(w, "    ", mdb)
}
