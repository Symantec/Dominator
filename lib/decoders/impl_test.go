package decoders

import (
	"testing"
)

type testdataType struct {
	Name  string
	Value string
}

func checkTestdata(t *testing.T, testdata testdataType) {
	if testdata.Name != "a name" {
		t.Fatalf("Name field: \"%s\" != \"a name\"", testdata.Name)
	}
	if testdata.Value != "a value" {
		t.Fatalf("Value field: \"%s\" != \"a value\"", testdata.Value)
	}
}

func TestGobExplicit(t *testing.T) {
	var testdata testdataType
	if err := DecodeFile("testdata/data.gob", &testdata); err != nil {
		t.Fatal(err)
	}
	checkTestdata(t, testdata)
}

func TestGobImplicit(t *testing.T) {
	var testdata testdataType
	if err := FindAndDecodeFile("testdata/data-gob", &testdata); err != nil {
		t.Fatal(err)
	}
	checkTestdata(t, testdata)
}

func TestJsonExplicit(t *testing.T) {
	var testdata testdataType
	if err := DecodeFile("testdata/data.json", &testdata); err != nil {
		t.Fatal(err)
	}
	checkTestdata(t, testdata)
}

func TestJsonImplicit(t *testing.T) {
	var testdata testdataType
	if err := FindAndDecodeFile("testdata/data-json", &testdata); err != nil {
		t.Fatal(err)
	}
	checkTestdata(t, testdata)
}
