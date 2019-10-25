package mdb

import (
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/json"
)

func (mdb *Mdb) debugWrite(w io.Writer) error {
	return json.WriteWithIndent(w, "    ", mdb)
}
