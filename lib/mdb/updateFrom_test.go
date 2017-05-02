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
				tagsType{"k0": "v0"}},
			&AwsMetadata{"aId-0", "aName-0", "i-2", "r-0",
				tagsType{"k0": "v0"}},
			true,
		},
		{ // One tag, different.
			&AwsMetadata{"aId-0", "aName-0", "i-3", "r-0",
				tagsType{"k0": "v0"}},
			&AwsMetadata{"aId-0", "aName-0", "i-3", "r-0",
				tagsType{"k0": "v1"}},
			false,
		},
		{ // Two tags, the same.
			&AwsMetadata{"aId-0", "aName-0", "i-4", "r-0",
				tagsType{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"aId-0", "aName-0", "i-4", "r-0",
				tagsType{"k0": "v0", "k1": "v1"}},
			true,
		},
		{ // Two tags added in a different order, the same
			&AwsMetadata{"aId-0", "aName-0", "i-5", "r-0",
				tagsType{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"aId-0", "aName-0", "i-5", "r-0",
				tagsType{"k1": "v1", "k0": "v0"}},
			true,
		},
		{ // Two tags, values swapped.
			&AwsMetadata{"aId-0", "aName-0", "i-6", "r-0",
				tagsType{"k0": "v0", "k1": "v1"}},
			&AwsMetadata{"aId-0", "aName-0", "i-6", "r-0",
				tagsType{"k0": "v1", "k1": "v0"}},
			false,
		},
	}
	for _, test := range tests {
		if got := compareAwsMetadata(test.left, test.right); got != test.want {
			t.Errorf("Less(%q, %q) = %v", test.left, test.right, got)
		}
	}
}
