package mdb

import (
	"io"
)

type Machine struct {
	Hostname       string
	RequiredImage  string `json:",omitempty"`
	PlannedImage   string `json:",omitempty"`
	DisableUpdates bool   `json:",omitempty"`
	OwnerGroup     string `json:",omitempty"`
}

func (dest *Machine) UpdateFrom(source Machine) {
	dest.updateFrom(source)
}

type Mdb struct {
	Machines []Machine
}

func (mdb *Mdb) DebugWrite(w io.Writer) error {
	return mdb.debugWrite(w)
}

func (mdb *Mdb) Len() int {
	return len(mdb.Machines)
}

func (mdb *Mdb) Less(left, right int) bool {
	if mdb.Machines[left].Hostname < mdb.Machines[right].Hostname {
		return true
	}
	return false
}

func (mdb *Mdb) Swap(left, right int) {
	tmp := mdb.Machines[left]
	mdb.Machines[left] = mdb.Machines[right]
	mdb.Machines[right] = tmp
}
