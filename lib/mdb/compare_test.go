package mdb

import (
	"reflect"
	"testing"

	"github.com/Cloud-Foundations/Dominator/lib/tags"
)

func makeNonzeroMachine(t *testing.T) Machine {
	var machine Machine
	machineValue := reflect.ValueOf(&machine).Elem()
	machineType := reflect.TypeOf(machine)
	for index := 0; index < machineValue.NumField(); index++ {
		fieldValue := machineValue.Field(index)
		fieldKind := fieldValue.Kind()
		switch fieldKind {
		case reflect.Bool:
			fieldValue.SetBool(true)
		case reflect.String:
			fieldValue.SetString(machineType.Field(index).Name)
		case reflect.Ptr:
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		case reflect.Map:
			mapValue := reflect.MakeMap(fieldValue.Type())
			fieldValue.Set(mapValue)
			mapValue.SetMapIndex(reflect.ValueOf("key"),
				reflect.ValueOf("value"))
		default:
			t.Fatalf("Unsupported field type: %s", fieldKind)
		}
	}
	return machine
}

func TestCompare(t *testing.T) {
	left := makeNonzeroMachine(t)
	right := Machine{Hostname: left.Hostname}
	if got := left.Compare(right); got != false {
		t.Errorf("Compare(%v, %v) = %v", left, right, got)
	}
	right = makeNonzeroMachine(t)
	if got := left.Compare(right); got != true {
		t.Errorf("Compare(%v, %v) = %v", left, right, got)
	}
	right.Tags["key"] = "value"
	if got := left.Compare(right); got != true {
		t.Errorf("Compare(%v, %v) = %v", left, right, got)
	}
	right.Tags["key"] = "another value"
	if got := left.Compare(right); got != false {
		t.Errorf("Compare(%v, %v) = %v", left, right, got)
	}
}

func TestCompareAwsMetadata(t *testing.T) {
	var tests = []struct {
		left, right *AwsMetadata
		want        bool
	}{
		{ // No tags, same metadata.
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-0", nil},
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-0", nil}, true},
		{ // No tags, different instance.
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-0", nil},
			&AwsMetadata{"aId-0", "aName-0", "i-1", "r-0", nil}, false},
		{ // No tags, different AccountId.
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-0", nil},
			&AwsMetadata{"aId-1", "aName-0", "i-0", "r-0", nil}, false},
		{ // No tags, different AccountName.
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-0", nil},
			&AwsMetadata{"aId-0", "aName-1", "i-0", "r-0", nil}, false},
		{ // No tags, different Region.
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-0", nil},
			&AwsMetadata{"aId-0", "aName-0", "i-0", "r-1", nil}, false},
		{ // One tag, the same.
			&AwsMetadata{"aId-0", "aName-0", "i-2", "r-0",
				tags.Tags{"k0": "v0"}},
			&AwsMetadata{"aId-0", "aName-0", "i-2", "r-0",
				tags.Tags{"k0": "v0"}},
			true,
		},
		{ // One tag, different.
			&AwsMetadata{"aId-0", "aName-0", "i-3", "r-0",
				tags.Tags{"k0": "v0"}},
			&AwsMetadata{"aId-0", "aName-0", "i-3", "r-0",
				tags.Tags{"k0": "v1"}},
			false,
		},
		{ // Two tags, the same.
			&AwsMetadata{"aId-0", "aName-0", "i-4", "r-0",
				tags.Tags{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"aId-0", "aName-0", "i-4", "r-0",
				tags.Tags{"k0": "v0", "k1": "v1"}},
			true,
		},
		{ // Two tags added in a different order, the same
			&AwsMetadata{"aId-0", "aName-0", "i-5", "r-0",
				tags.Tags{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"aId-0", "aName-0", "i-5", "r-0",
				tags.Tags{"k1": "v1", "k0": "v0"}},
			true,
		},
		{ // Two tags, values swapped.
			&AwsMetadata{"aId-0", "aName-0", "i-6", "r-0",
				tags.Tags{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"aId-0", "aName-0", "i-6", "r-0",
				tags.Tags{"k0": "v1", "k1": "v0"}},
			false,
		},
	}
	for _, test := range tests {
		if got := compareAwsMetadata(test.left, test.right); got != test.want {
			t.Errorf("Less(%q, %q) = %v", test.left, test.right, got)
		}
	}
}
