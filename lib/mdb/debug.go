package mdb

import (
	"io"

	"github.com/Symantec/Dominator/lib/json"
)

func (mdb *Mdb) debugWrite(w io.Writer) error {
	return json.WriteWithIndent(w, "    ", mdb)
}
