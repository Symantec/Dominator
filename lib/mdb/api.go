/*
	Package mdb implements a simple in-memory Machine DataBase.
*/
package mdb

import (
	"io"

	"github.com/Symantec/Dominator/lib/tags"
	"github.com/Symantec/Dominator/lib/verstr"
)

type AwsMetadata struct {
	AccountId   string
	AccountName string
	InstanceId  string
	Region      string
	Tags        tags.Tags
}

// Machine describes a single machine with a unique Hostname and optional
// metadata about the machine.
type Machine struct {
	Hostname       string
	IpAddress      string       `json:",omitempty"`
	RequiredImage  string       `json:",omitempty"`
	PlannedImage   string       `json:",omitempty"`
	DisableUpdates bool         `json:",omitempty"`
	OwnerGroup     string       `json:",omitempty"`
	Tags           tags.Tags    `json:",omitempty"`
	AwsMetadata    *AwsMetadata `json:",omitempty"`
}

func (left Machine) Compare(right Machine) bool {
	return left.compare(right)
}

// UpdateFrom updates dest with data from source.
func (dest *Machine) UpdateFrom(source Machine) {
	dest.updateFrom(source)
}

// Mdb describes a list of Machines. It implements sort.Interface.
type Mdb struct {
	Machines []Machine
}

// DebugWrite writes the JSON representation to w.
func (mdb *Mdb) DebugWrite(w io.Writer) error {
	return mdb.debugWrite(w)
}

// Len returns the number of machines.
func (mdb *Mdb) Len() int {
	return len(mdb.Machines)
}

// Less compares the hostnames of left and right.
func (mdb *Mdb) Less(left, right int) bool {
	return verstr.Less(mdb.Machines[left].Hostname,
		mdb.Machines[right].Hostname)
}

// Swap swaps two entries in mdb.
func (mdb *Mdb) Swap(left, right int) {
	tmp := mdb.Machines[left]
	mdb.Machines[left] = mdb.Machines[right]
	mdb.Machines[right] = tmp
}
