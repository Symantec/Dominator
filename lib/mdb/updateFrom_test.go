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
		case reflect.Ptr:
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
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

func TestCompareAwsMetadata(t *testing.T) {
	type tagsType map[string]string
	var tests = []struct {
		left, right *AwsMetadata
		want        bool
	}{
		{&AwsMetadata{"i-0", nil}, &AwsMetadata{"i-0", nil}, true},
		{&AwsMetadata{"i-0", nil}, &AwsMetadata{"i-1", nil}, false},
		{ // One tag, the same.
			&AwsMetadata{"i-2", tagsType{"k0": "v0"}},
			&AwsMetadata{"i-2", tagsType{"k0": "v0"}},
			true,
		},
		{ // One tag, different.
			&AwsMetadata{"i-3", tagsType{"k0": "v0"}},
			&AwsMetadata{"i-3", tagsType{"k0": "v1"}},
			false,
		},
		{ // Two tags, the same.
			&AwsMetadata{"i-4", tagsType{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"i-4", tagsType{"k0": "v0", "k1": "v1"}},
			true,
		},
		{ // Two tags added in a different order, the same
			&AwsMetadata{"i-5", tagsType{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"i-5", tagsType{"k1": "v1", "k0": "v0"}},
			true,
		},
		{ // Two tags, values swapped.
			&AwsMetadata{"i-6", tagsType{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"i-6", tagsType{"k0": "v1", "k1": "v0"}},
			false,
		},
	}
	for _, test := range tests {
		if got := compareAwsMetadata(test.left, test.right); got != test.want {
			t.Errorf("Less(%q, %q) = %v", test.left, test.right, got)
		}
	}
}
