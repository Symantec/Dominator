package mdb

import (
	"testing"
)

func TestUpdateFrom(t *testing.T) {
	source := makeNonzeroMachine(t)
	dest := &Machine{Hostname: "some.host"}
	dest.UpdateFrom(source)
	defaultMachine := &Machine{Hostname: "some.host"}
	if !dest.Compare(*defaultMachine) {
		t.Errorf("UpdateFrom(): copied data despite Hostname mismatch: %v",
			*dest)
	}
	dest.Hostname = "Hostname"
	dest.UpdateFrom(source)
	if !dest.Compare(source) {
		t.Errorf("UpdateFrom: %v != %v", *dest, source)
	}
}
