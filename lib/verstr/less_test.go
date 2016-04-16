package verstr

import (
	"testing"
)

func TestLess(t *testing.T) {
	var tests = []struct {
		left, right string
		want        bool
	}{
		{"file.0.ext", "file.1.ext", true},
		{"file.1.ext", "file.0.ext", false},
		{"file.1.ext", "file.10.ext", true},
		{"file.10.ext", "file.1.ext", false},
		{"file.9.ext", "file.10.ext", true},
		{"file.10.ext", "file.9.ext", false},
		{"sparse", "sparse.0", true},
		{"sparse.0", "sparse", false},
	}
	for _, test := range tests {
		if got := Less(test.left, test.right); got != test.want {
			t.Errorf("Less(%q, %q) = %v", test.left, test.right, got)
		}
	}
}
