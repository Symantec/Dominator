package mdb

import (
	"reflect"
	"testing"
)

func TestUpdateFrom(t *testing.T) {
	var source Machine
	sourceValue := reflect.ValueOf(&source).Elem()
	sourceType := reflect.TypeOf(source)
	for index := 0; index < sourceValue.NumField(); index++ {
		fieldValue := sourceValue.Field(index)
		fieldKind := fieldValue.Kind()
		switch fieldKind {
		case reflect.Bool:
			fieldValue.SetBool(true)
		case reflect.String:
			fieldValue.SetString(sourceType.Field(index).Name)
		default:
			t.Errorf("Unsupported field type: %s", fieldKind)
		}
	}
	dest := &Machine{Hostname: "some.host"}
	dest.UpdateFrom(source)
	defaultMachine := &Machine{Hostname: "some.host"}
	if *dest != *defaultMachine {
		t.Errorf("UpdateFrom(): copied data despite Hostname mismatch: %v",
			*dest)
	}
	dest.Hostname = "Hostname"
	dest.UpdateFrom(source)
	if *dest != source {
		t.Errorf("UpdateFrom: %v != %v", *dest, source)
	}
}
